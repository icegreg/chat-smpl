import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Chat, Message, Participant, CreateChatRequest, SendMessageRequest, CentrifugoEvent } from '@/types'
import { api, ApiError } from '@/api/client'
import { Centrifuge } from 'centrifuge'

export const useChatStore = defineStore('chat', () => {
  const chats = ref<Chat[]>([])
  const currentChat = ref<Chat | null>(null)
  const messages = ref<Message[]>([])
  const participants = ref<Participant[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const typingUsers = ref<Map<string, Set<string>>>(new Map())

  let centrifuge: Centrifuge | null = null
  const subscriptions = new Map<string, unknown>()

  const sortedChats = computed(() => {
    return [...chats.value].sort((a, b) => {
      // Favorites first
      if (a.is_favorite && !b.is_favorite) return -1
      if (!a.is_favorite && b.is_favorite) return 1
      // Then by last message time
      const aTime = a.last_message?.created_at || a.created_at
      const bTime = b.last_message?.created_at || b.created_at
      return new Date(bTime).getTime() - new Date(aTime).getTime()
    })
  })

  async function initCentrifuge() {
    if (centrifuge) return

    try {
      const { token } = await api.getCentrifugoConnectionToken()

      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${wsProtocol}//${window.location.host}/connection/websocket`
      centrifuge = new Centrifuge(wsUrl, {
        token,
        getToken: async () => {
          const { token } = await api.getCentrifugoConnectionToken()
          return token
        },
      })

      centrifuge.on('connected', () => {
        console.log('Connected to Centrifugo')
      })

      centrifuge.on('disconnected', () => {
        console.log('Disconnected from Centrifugo')
      })

      centrifuge.connect()
    } catch (e) {
      console.error('Failed to initialize Centrifugo', e)
    }
  }

  function subscribeToChatChannel(chatId: string) {
    if (!centrifuge || subscriptions.has(chatId)) return

    const channel = `chat:${chatId}`
    const sub = centrifuge.newSubscription(channel, {
      getToken: async () => {
        const { token } = await api.getCentrifugoSubscriptionToken(channel)
        return token
      },
    })

    sub.on('publication', (ctx) => {
      handleCentrifugoEvent(ctx.data as CentrifugoEvent)
    })

    sub.subscribe()
    subscriptions.set(chatId, sub)
  }

  function unsubscribeFromChatChannel(chatId: string) {
    const sub = subscriptions.get(chatId) as { unsubscribe: () => void } | undefined
    if (sub) {
      sub.unsubscribe()
      subscriptions.delete(chatId)
    }
  }

  function handleCentrifugoEvent(event: CentrifugoEvent) {
    switch (event.type) {
      case 'message.new':
        handleNewMessage(event.data as Message)
        break
      case 'message.update':
        handleMessageUpdate(event.data as Message)
        break
      case 'message.delete':
        handleMessageDelete((event.data as { message_id: string }).message_id)
        break
      case 'typing':
        handleTyping(event.chat_id!, event.user_id!, (event.data as { is_typing: boolean }).is_typing)
        break
      case 'chat.update':
        handleChatUpdate(event.data as Chat)
        break
    }
  }

  function handleNewMessage(message: Message) {
    if (currentChat.value?.id === message.chat_id) {
      messages.value.push(message)
    }
    // Update last message in chat list
    const chat = chats.value.find((c) => c.id === message.chat_id)
    if (chat) {
      chat.last_message = message
      if (currentChat.value?.id !== message.chat_id) {
        chat.unread_count = (chat.unread_count || 0) + 1
      }
    }
  }

  function handleMessageUpdate(message: Message) {
    const index = messages.value.findIndex((m) => m.id === message.id)
    if (index !== -1) {
      messages.value[index] = message
    }
  }

  function handleMessageDelete(messageId: string) {
    const index = messages.value.findIndex((m) => m.id === messageId)
    if (index !== -1) {
      messages.value.splice(index, 1)
    }
  }

  function handleTyping(chatId: string, userId: string, isTyping: boolean) {
    if (!typingUsers.value.has(chatId)) {
      typingUsers.value.set(chatId, new Set())
    }
    const users = typingUsers.value.get(chatId)!
    if (isTyping) {
      users.add(userId)
    } else {
      users.delete(userId)
    }
  }

  function handleChatUpdate(chat: Chat) {
    const index = chats.value.findIndex((c) => c.id === chat.id)
    if (index !== -1) {
      chats.value[index] = { ...chats.value[index], ...chat }
    }
    if (currentChat.value?.id === chat.id) {
      currentChat.value = { ...currentChat.value, ...chat }
    }
  }

  async function fetchChats() {
    loading.value = true
    error.value = null
    try {
      const result = await api.getChats()
      chats.value = result.chats || []
      // Subscribe to all chat channels
      for (const chat of chats.value) {
        subscribeToChatChannel(chat.id)
      }
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to fetch chats'
    } finally {
      loading.value = false
    }
  }

  async function selectChat(chatId: string) {
    loading.value = true
    error.value = null
    try {
      currentChat.value = await api.getChat(chatId)
      const messagesResult = await api.getMessages(chatId)
      messages.value = messagesResult.messages || []
      const participantsResult = await api.getParticipants(chatId)
      participants.value = participantsResult.participants || []
      // Clear unread count
      const chat = chats.value.find((c) => c.id === chatId)
      if (chat) {
        chat.unread_count = 0
      }
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to load chat'
    } finally {
      loading.value = false
    }
  }

  async function createChat(data: CreateChatRequest) {
    loading.value = true
    error.value = null
    try {
      const chat = await api.createChat(data)
      chats.value.unshift(chat)
      subscribeToChatChannel(chat.id)
      return chat
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to create chat'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function sendMessage(data: SendMessageRequest) {
    if (!currentChat.value) return
    error.value = null
    try {
      const message = await api.sendMessage(currentChat.value.id, data)
      // Message will be added via Centrifugo event
      return message
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to send message'
      throw e
    }
  }

  async function updateMessage(messageId: string, content: string) {
    error.value = null
    try {
      await api.updateMessage(messageId, content)
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to update message'
      throw e
    }
  }

  async function deleteMessage(messageId: string) {
    error.value = null
    try {
      await api.deleteMessage(messageId)
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to delete message'
      throw e
    }
  }

  async function addReaction(messageId: string, emoji: string) {
    try {
      await api.addReaction(messageId, emoji)
    } catch (e) {
      console.error('Failed to add reaction', e)
    }
  }

  async function removeReaction(messageId: string, emoji: string) {
    try {
      await api.removeReaction(messageId, emoji)
    } catch (e) {
      console.error('Failed to remove reaction', e)
    }
  }

  async function toggleFavorite(chatId: string) {
    const chat = chats.value.find((c) => c.id === chatId)
    if (!chat) return
    try {
      if (chat.is_favorite) {
        await api.removeFromFavorites(chatId)
        chat.is_favorite = false
      } else {
        await api.addToFavorites(chatId)
        chat.is_favorite = true
      }
    } catch (e) {
      console.error('Failed to toggle favorite', e)
    }
  }

  async function sendTyping(isTyping: boolean) {
    if (!currentChat.value) return
    try {
      await api.sendTypingIndicator(currentChat.value.id, isTyping)
    } catch {
      // Ignore typing errors
    }
  }

  function cleanup() {
    if (centrifuge) {
      centrifuge.disconnect()
      centrifuge = null
    }
    subscriptions.clear()
    chats.value = []
    currentChat.value = null
    messages.value = []
    participants.value = []
  }

  return {
    chats,
    currentChat,
    messages,
    participants,
    loading,
    error,
    typingUsers,
    sortedChats,
    initCentrifuge,
    fetchChats,
    selectChat,
    createChat,
    sendMessage,
    updateMessage,
    deleteMessage,
    addReaction,
    removeReaction,
    toggleFavorite,
    sendTyping,
    cleanup,
    unsubscribeFromChatChannel,
  }
})
