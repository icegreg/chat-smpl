import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Chat, Message, Participant, CreateChatRequest, SendMessageRequest } from '@/types'
import { api, ApiError } from '@/api/client'
import { Centrifuge, Subscription } from 'centrifuge'
import { useAuthStore } from './auth'

// Event types from websocket-service
interface ChatEvent {
  type: string
  timestamp: string
  actor_id: string
  chat_id: string
  data: unknown
}

export const useChatStore = defineStore('chat', () => {
  const chats = ref<Chat[]>([])
  const currentChat = ref<Chat | null>(null)
  const messages = ref<Message[]>([])
  const participants = ref<Participant[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const typingUsers = ref<Map<string, Set<string>>>(new Map())
  const typingTimeouts = ref<Map<string, number>>(new Map()) // chatId:userId -> timeout id

  let centrifuge: Centrifuge | null = null
  let userSubscription: Subscription | null = null

  const TYPING_DISPLAY_DURATION = 5000 // Hide typing indicator after 5 seconds

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

    const authStore = useAuthStore()
    if (!authStore.user) {
      console.error('Cannot init Centrifugo: user not authenticated')
      return
    }

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
        // Subscribe to user's personal channel
        subscribeToUserChannel(authStore.user!.id)
      })

      centrifuge.on('disconnected', () => {
        console.log('Disconnected from Centrifugo')
      })

      centrifuge.connect()
    } catch (e) {
      console.error('Failed to initialize Centrifugo', e)
    }
  }

  function subscribeToUserChannel(userId: string) {
    if (!centrifuge || userSubscription) return

    const channel = `user:${userId}`
    userSubscription = centrifuge.newSubscription(channel, {
      getToken: async () => {
        const { token } = await api.getCentrifugoSubscriptionToken(channel)
        return token
      },
    })

    userSubscription.on('publication', (ctx) => {
      handleCentrifugoEvent(ctx.data as ChatEvent)
    })

    userSubscription.subscribe()
    console.log('Subscribed to user channel:', channel)
  }

  function unsubscribeFromUserChannel() {
    if (userSubscription) {
      userSubscription.unsubscribe()
      userSubscription = null
    }
  }

  function handleCentrifugoEvent(event: ChatEvent) {
    console.log('Received event:', event.type, event)

    switch (event.type) {
      case 'message.created':
        handleNewMessage(event.data as Message)
        break
      case 'message.updated':
        handleMessageUpdate(event.data as Message)
        break
      case 'message.deleted':
        handleMessageDelete((event.data as { message_id: string }).message_id)
        break
      case 'typing':
        handleTyping(event.chat_id, event.actor_id, (event.data as { is_typing: boolean }).is_typing)
        break
      case 'chat.created':
        handleNewChat(event.data as Chat)
        break
      case 'chat.updated':
        handleChatUpdate(event.data as Chat)
        break
      case 'chat.deleted':
        handleChatDeleted(event.chat_id)
        break
      case 'reaction.added':
        handleReactionAdded(event.data as { message_id: string; emoji: string; user_id: string })
        break
      case 'reaction.removed':
        handleReactionRemoved(event.data as { message_id: string; emoji: string; user_id: string })
        break
    }
  }

  function handleNewMessage(message: Message) {
    if (currentChat.value?.id === message.chat_id) {
      // Avoid duplicates (message may already exist from REST response)
      const existing = messages.value.find((m) => m.id === message.id)
      if (!existing) {
        messages.value.push(message)
      }
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
    const timeoutKey = `${chatId}:${userId}`

    // Clear existing timeout for this user
    const existingTimeout = typingTimeouts.value.get(timeoutKey)
    if (existingTimeout) {
      clearTimeout(existingTimeout)
      typingTimeouts.value.delete(timeoutKey)
    }

    if (isTyping) {
      users.add(userId)

      // Set timeout to auto-remove typing indicator after 5 seconds
      const timeout = window.setTimeout(() => {
        users.delete(userId)
        typingTimeouts.value.delete(timeoutKey)
      }, TYPING_DISPLAY_DURATION)
      typingTimeouts.value.set(timeoutKey, timeout)
    } else {
      users.delete(userId)
    }
  }

  function handleNewChat(chatData: Chat) {
    // Add new chat to the list if not already there
    const exists = chats.value.some((c) => c.id === chatData.id)
    if (!exists) {
      chats.value.unshift(chatData)
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

  function handleChatDeleted(chatId: string) {
    const index = chats.value.findIndex((c) => c.id === chatId)
    if (index !== -1) {
      chats.value.splice(index, 1)
    }
    if (currentChat.value?.id === chatId) {
      currentChat.value = null
      messages.value = []
      participants.value = []
    }
  }

  function handleReactionAdded(data: { message_id: string; emoji: string; user_id: string }) {
    const message = messages.value.find((m) => m.id === data.message_id)
    if (message) {
      if (!message.reactions) {
        message.reactions = []
      }
      // Find existing reaction group for this emoji
      const existingReaction = message.reactions.find((r) => r.emoji === data.emoji)
      if (existingReaction) {
        // Add user to existing reaction if not already there
        if (!existingReaction.users.includes(data.user_id)) {
          existingReaction.users.push(data.user_id)
          existingReaction.count++
        }
      } else {
        // Create new reaction group
        message.reactions.push({
          emoji: data.emoji,
          count: 1,
          users: [data.user_id],
        })
      }
    }
  }

  function handleReactionRemoved(data: { message_id: string; emoji: string; user_id: string }) {
    const message = messages.value.find((m) => m.id === data.message_id)
    if (message && message.reactions) {
      const reactionIndex = message.reactions.findIndex((r) => r.emoji === data.emoji)
      if (reactionIndex !== -1) {
        const reaction = message.reactions[reactionIndex]
        const userIndex = reaction.users.indexOf(data.user_id)
        if (userIndex !== -1) {
          reaction.users.splice(userIndex, 1)
          reaction.count--
          // Remove reaction group if no users left
          if (reaction.count === 0) {
            message.reactions.splice(reactionIndex, 1)
          }
        }
      }
    }
  }

  async function fetchChats() {
    loading.value = true
    error.value = null
    try {
      const result = await api.getChats()
      chats.value = result.chats || []
      // No need to subscribe to individual chat channels anymore
      // Events come through user's personal channel
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
      // Chat events will come via user channel, no per-chat subscription needed
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
      // Add message immediately using REST response (includes file_attachments)
      // The Centrifugo event may have already added this message WITHOUT file_attachments
      // so we need to update/replace it with the enriched version
      const existingIndex = messages.value.findIndex(m => m.id === message.id)
      if (existingIndex !== -1) {
        // Replace existing message with enriched one (has file_attachments)
        messages.value[existingIndex] = message
      } else {
        messages.value.push(message)
      }
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
    unsubscribeFromUserChannel()
    if (centrifuge) {
      centrifuge.disconnect()
      centrifuge = null
    }
    chats.value = []
    currentChat.value = null
    messages.value = []
    participants.value = []
    typingUsers.value.clear()
    // Clear all typing timeouts
    typingTimeouts.value.forEach((timeout) => clearTimeout(timeout))
    typingTimeouts.value.clear()
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
  }
})
