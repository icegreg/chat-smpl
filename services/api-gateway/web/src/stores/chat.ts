import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Chat, Message, Participant, CreateChatRequest, SendMessageRequest } from '@/types'
import { api, ApiError } from '@/api/client'
import { Centrifuge, Subscription } from 'centrifuge'
import { useAuthStore } from './auth'
import { usePresenceStore } from './presence'
import { useNetworkStore } from './network'
import { useVoiceStore } from './voice'

// Event types from websocket-service
interface ChatEvent {
  type: string
  timestamp: string
  actor_id: string
  chat_id: string
  data: unknown
}

// Storage key for seq_num persistence
const SEQ_NUM_STORAGE_KEY = 'chat_seq_nums'

export const useChatStore = defineStore('chat', () => {
  const chats = ref<Chat[]>([])
  const currentChat = ref<Chat | null>(null)
  const messages = ref<Message[]>([])
  const participants = ref<Participant[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const typingUsers = ref<Map<string, Set<string>>>(new Map())
  const typingTimeouts = ref<Map<string, number>>(new Map()) // chatId:userId -> timeout id

  // Cursor-based pagination for chats
  const chatsCursor = ref<string>('')
  const chatsHasMore = ref(false)
  const chatsTotal = ref(0)
  const loadingMoreChats = ref(false)

  // Track last known seq_num per chat for reliable sync after reconnect
  const lastSeqNums = ref<Map<string, number>>(new Map())
  const isSyncing = ref(false)

  let centrifuge: Centrifuge | null = null
  let userSubscription: Subscription | null = null

  const TYPING_DISPLAY_DURATION = 5000 // Hide typing indicator after 5 seconds

  // Load seq_nums from localStorage on init
  function loadSeqNumsFromStorage() {
    try {
      const stored = localStorage.getItem(SEQ_NUM_STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored) as Record<string, number>
        Object.entries(parsed).forEach(([chatId, seqNum]) => {
          lastSeqNums.value.set(chatId, seqNum)
        })
      }
    } catch (e) {
      console.warn('Failed to load seq_nums from storage:', e)
    }
  }

  // Save seq_nums to localStorage
  function saveSeqNumsToStorage() {
    try {
      const obj: Record<string, number> = {}
      lastSeqNums.value.forEach((seqNum, chatId) => {
        obj[chatId] = seqNum
      })
      localStorage.setItem(SEQ_NUM_STORAGE_KEY, JSON.stringify(obj))
    } catch (e) {
      console.warn('Failed to save seq_nums to storage:', e)
    }
  }

  // Update seq_num tracking when we receive/load messages
  function updateLastSeqNum(chatId: string, seqNum: number | undefined) {
    if (seqNum === undefined) return
    const current = lastSeqNums.value.get(chatId) || 0
    if (seqNum > current) {
      lastSeqNums.value.set(chatId, seqNum)
      saveSeqNumsToStorage()
    }
  }

  // Load seq_nums on module init
  loadSeqNumsFromStorage()

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
        // Reconnect settings for fault tolerance
        minReconnectDelay: 500,      // Start with 500ms delay
        maxReconnectDelay: 20000,    // Max 20 seconds between attempts
        timeout: 10000,              // Connection timeout 10s
        maxServerPingDelay: 15000,   // Detect stale connection after 15s
      })

      centrifuge.on('connected', () => {
        console.log('Connected to Centrifugo')
        // Update network store
        const networkStore = useNetworkStore()
        networkStore.setWebSocketConnected(true)

        // Subscribe to user's personal channel
        // Recovery will happen automatically via subscription's 'subscribed' event
        subscribeToUserChannel(authStore.user!.id)

        // Process any pending messages
        networkStore.processPendingMessages()

        // Register presence connection in background with delay
        // This is low priority and can be slow - delay to not block other requests
        setTimeout(() => {
          const presenceStore = usePresenceStore()
          presenceStore.registerConnection().catch(e => {
            console.warn('Presence registration failed:', e)
          })
        }, 2000) // 2 second delay
      })

      centrifuge.on('disconnected', (ctx) => {
        console.log('Disconnected from Centrifugo:', ctx.reason)
        // Update network store
        const networkStore = useNetworkStore()
        networkStore.setWebSocketConnected(false)

        // Unregister presence connection
        const presenceStore = usePresenceStore()
        presenceStore.unregisterConnection()
      })

      centrifuge.on('connecting', (ctx) => {
        console.log('Reconnecting to Centrifugo...', ctx.reason)
      })

      centrifuge.on('error', (ctx) => {
        console.error('Centrifugo error:', ctx.error)
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
      // Enable recovery to get missed messages after reconnect
      // Centrifugo will automatically send messages missed while offline
      recoverable: true,
    })

    userSubscription.on('publication', (ctx) => {
      handleCentrifugoEvent(ctx.data as ChatEvent)
    })

    // Handle recovered publications (messages received while offline)
    // Note: Recovered messages are delivered automatically via 'publication' events
    // after the 'subscribed' event, so we just need to check recovery status
    userSubscription.on('subscribed', (ctx) => {
      console.log('Subscribed to user channel:', channel)
      console.log('  wasRecovering:', ctx.wasRecovering, 'recovered:', ctx.recovered)

      if (ctx.wasRecovering && ctx.recovered) {
        // Recovery successful - missed messages will arrive via 'publication' events
        console.log('Recovery successful - missed messages will be delivered automatically')
      } else if (ctx.wasRecovering && !ctx.recovered) {
        // Recovery failed - need to sync via REST API as fallback
        console.warn('Recovery failed, syncing via API...')
        syncAllChatsAfterReconnect()
      }
    })

    userSubscription.subscribe()
    console.log('Subscribing to user channel:', channel)
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
        handleMessageDelete(event.data as { message_id: string; chat_id: string; is_moderated_deletion?: boolean })
        break
      case 'message.restored':
        handleMessageRestored(event.data as Message)
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

      // Voice events - forward to voice store
      case 'conference.created':
      case 'conference.ended':
      case 'participant.joined':
      case 'participant.left':
      case 'call.initiated':
      case 'call.answered':
      case 'call.ended':
        {
          const voiceStore = useVoiceStore()
          voiceStore.handleVoiceEvent({ type: event.type, data: event.data })
        }
        break
    }
  }

  function handleNewMessage(message: Message) {
    // Update seq_num tracking
    updateLastSeqNum(message.chat_id, message.seq_num)

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

  function handleMessageDelete(data: { message_id: string; chat_id: string; is_moderated_deletion?: boolean }) {
    const index = messages.value.findIndex((m) => m.id === data.message_id)
    if (index !== -1) {
      // Mark as deleted instead of removing (soft delete)
      messages.value[index] = {
        ...messages.value[index],
        is_deleted: true,
        is_moderated_deletion: data.is_moderated_deletion || false,
        content: '', // Clear content for privacy
        deleted_at: new Date().toISOString(),
      }
    }
  }

  function handleMessageRestored(message: Message) {
    const index = messages.value.findIndex((m) => m.id === message.id)
    if (index !== -1) {
      // Update message with restored data
      messages.value[index] = {
        ...messages.value[index],
        ...message,
        is_deleted: false,
        is_moderated_deletion: false,
        deleted_at: undefined,
        deleted_by: undefined,
      }
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
    // Reset cursor pagination state
    chatsCursor.value = ''
    chatsHasMore.value = false
    chatsTotal.value = 0
    try {
      const result = await api.getChats(50)
      chats.value = result.chats || []
      chatsCursor.value = result.next_cursor || ''
      chatsHasMore.value = result.has_more || false
      chatsTotal.value = result.total || 0
      // No need to subscribe to individual chat channels anymore
      // Events come through user's personal channel
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to fetch chats'
    } finally {
      loading.value = false
    }
  }

  async function loadMoreChats() {
    if (!chatsHasMore.value || loadingMoreChats.value || !chatsCursor.value) {
      return
    }
    loadingMoreChats.value = true
    error.value = null
    try {
      const result = await api.getChats(50, chatsCursor.value)
      const newChats = result.chats || []
      // Append to existing chats, avoiding duplicates
      for (const chat of newChats) {
        if (!chats.value.some(c => c.id === chat.id)) {
          chats.value.push(chat)
        }
      }
      chatsCursor.value = result.next_cursor || ''
      chatsHasMore.value = result.has_more || false
      // Total stays the same or could be updated
      if (result.total) {
        chatsTotal.value = result.total
      }
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to load more chats'
    } finally {
      loadingMoreChats.value = false
    }
  }

  async function selectChat(chatId: string) {
    loading.value = true
    error.value = null
    try {
      currentChat.value = await api.getChat(chatId)
      const messagesResult = await api.getMessages(chatId)
      messages.value = messagesResult.messages || []

      // Update lastSeqNum from loaded messages
      if (messages.value.length > 0) {
        const maxSeqNum = Math.max(...messages.value.map((m) => m.seq_num || 0))
        if (maxSeqNum > 0) {
          updateLastSeqNum(chatId, maxSeqNum)
        }
      }

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
      // Don't add chat here - it will be added via 'chat.created' event from WebSocket
      // This prevents duplicate chats appearing in the list
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

    const networkStore = useNetworkStore()
    const chatId = currentChat.value.id

    try {
      const message = await api.sendMessage(chatId, data)
      // Update seq_num tracking from sent message
      updateLastSeqNum(chatId, message.seq_num)

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
      const apiError = e instanceof ApiError ? e : null

      // If it's a network error, add to pending queue
      if (apiError?.isNetworkError || !networkStore.isOnline) {
        console.log('[ChatStore] Network error, adding message to pending queue')
        const pendingId = networkStore.addPendingMessage(
          chatId,
          data.content,
          data.file_link_ids
        )

        // Create a temporary pending message for UI
        const pendingMessage: Message = {
          id: pendingId,
          chat_id: chatId,
          sender_id: useAuthStore().user?.id || '',
          content: data.content,
          created_at: new Date().toISOString(),
          is_pending: true, // Special flag for UI
        }
        messages.value.push(pendingMessage)

        // Don't throw - message will be sent when connection restores
        return pendingMessage
      }

      error.value = apiError?.message || 'Failed to send message'
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

  async function restoreMessage(messageId: string) {
    error.value = null
    try {
      const message = await api.restoreMessage(messageId)
      // Update local state
      const index = messages.value.findIndex((m) => m.id === messageId)
      if (index !== -1) {
        messages.value[index] = message
      }
      return message
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to restore message'
      throw e
    }
  }

  async function removeFromQuote(messageId: string, quotedMessageId: string) {
    error.value = null
    try {
      await api.removeFromQuote(messageId, quotedMessageId)
      // Update local state - remove the quoted message reference
      const index = messages.value.findIndex((m) => m.id === messageId)
      if (index !== -1) {
        const msg = messages.value[index]
        if (msg.reply_to_ids) {
          msg.reply_to_ids = msg.reply_to_ids.filter(id => id !== quotedMessageId)
        }
        if (msg.reply_to_messages) {
          msg.reply_to_messages = msg.reply_to_messages.filter(m => m.id !== quotedMessageId)
        }
      }
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to remove from quote'
      throw e
    }
  }

  async function forwardMessage(
    message: Message,
    sourceChatId: string,
    targetChatId: string,
    comment?: string
  ) {
    error.value = null
    try {
      // Build forwarded message content
      const forwardedContent = comment
        ? `${comment}\n\n↪️ Forwarded from ${message.sender_display_name || message.sender_username || 'unknown'}:\n${message.content}`
        : message.content

      // Send message to target chat with forwarded info
      const newMessage = await api.sendMessage(targetChatId, {
        content: forwardedContent,
        forwarded_from_id: message.id,
        forwarded_from_chat_id: sourceChatId,
        // Also forward file attachments if any
        file_link_ids: message.file_attachments?.map(f => f.link_id)
      })

      return newMessage
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Failed to forward message'
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

  async function deleteChat(chatId: string) {
    error.value = null
    try {
      await api.deleteChat(chatId)
      // handleChatDeleted will be called via WebSocket event
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Не удалось удалить чат'
      throw e
    }
  }

  async function addParticipant(chatId: string, userId: string, role = 'member') {
    error.value = null
    try {
      await api.addParticipant(chatId, userId, role)
      // Refresh participants list
      const result = await api.getParticipants(chatId)
      participants.value = result.participants || []
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Не удалось добавить участника'
      throw e
    }
  }

  async function removeParticipant(chatId: string, userId: string) {
    error.value = null
    try {
      await api.removeParticipant(chatId, userId)
      // Remove from local state
      participants.value = participants.value.filter(p => p.user_id !== userId)
    } catch (e) {
      error.value = e instanceof ApiError ? e.message : 'Не удалось удалить участника'
      throw e
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

  // Sync messages after reconnect - fetch any messages we missed while offline
  async function syncMessagesAfterReconnect(chatId: string) {
    if (isSyncing.value) return

    const lastSeqNum = lastSeqNums.value.get(chatId)
    if (lastSeqNum === undefined || lastSeqNum === 0) {
      console.log('No seq_num tracked for chat, skipping sync:', chatId)
      return
    }

    isSyncing.value = true
    console.log(`Syncing messages for chat ${chatId} after seq_num ${lastSeqNum}`)

    try {
      let hasMore = true
      let afterSeq = lastSeqNum

      while (hasMore) {
        const result = await api.syncMessages(chatId, afterSeq, 100)
        const newMessages = result.messages || []
        hasMore = result.has_more

        if (newMessages.length === 0) break

        // Add new messages to the list, avoiding duplicates
        for (const msg of newMessages) {
          const existing = messages.value.find((m) => m.id === msg.id)
          if (!existing) {
            messages.value.push(msg)
          }
          // Update seq_num tracking
          updateLastSeqNum(chatId, msg.seq_num)
          afterSeq = msg.seq_num || afterSeq
        }

        console.log(`Synced ${newMessages.length} messages, has_more: ${hasMore}`)
      }

      // Sort messages by seq_num or created_at to ensure correct order
      messages.value.sort((a, b) => {
        if (a.seq_num && b.seq_num) return a.seq_num - b.seq_num
        const aTime = new Date(a.created_at || 0).getTime()
        const bTime = new Date(b.created_at || 0).getTime()
        return aTime - bTime
      })

    } catch (e) {
      console.error('Failed to sync messages after reconnect:', e)
    } finally {
      isSyncing.value = false
    }
  }

  // Sync all chats after reconnect when Centrifugo recovery failed
  // This is a fallback to ensure we don't miss any messages
  async function syncAllChatsAfterReconnect() {
    console.log('Starting full sync for all chats...')

    // First refresh the chat list to get updated last_message and unread counts
    try {
      await fetchChats()
    } catch (e) {
      console.error('Failed to refresh chats:', e)
    }

    // Then sync messages for the current chat if open
    if (currentChat.value) {
      await syncMessagesAfterReconnect(currentChat.value.id)
    }

    // Sync other chats that have tracked seq_nums (user had them open before)
    // Do this in background to not block UI
    const otherChats = Array.from(lastSeqNums.value.keys()).filter(
      chatId => chatId !== currentChat.value?.id
    )

    for (const chatId of otherChats) {
      // Find the chat in our list
      const chat = chats.value.find(c => c.id === chatId)
      if (!chat) continue

      try {
        // Just check if there are new messages by comparing seq_num
        const result = await api.syncMessages(chatId, lastSeqNums.value.get(chatId) || 0, 1)
        if (result.messages && result.messages.length > 0) {
          // Update unread count indicator
          chat.unread_count = (chat.unread_count || 0) + result.messages.length
          if (result.has_more) {
            chat.unread_count += 1 // At least one more
          }
          // Update last message
          const lastMsg = result.messages[result.messages.length - 1]
          chat.last_message = lastMsg
          updateLastSeqNum(chatId, lastMsg.seq_num)
        }
      } catch (e) {
        console.warn(`Failed to check sync for chat ${chatId}:`, e)
      }
    }

    console.log('Full sync completed')
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
    isSyncing,
    // Cursor pagination for chats
    chatsCursor,
    chatsHasMore,
    chatsTotal,
    loadingMoreChats,
    initCentrifuge,
    fetchChats,
    loadMoreChats,
    selectChat,
    createChat,
    deleteChat,
    addParticipant,
    removeParticipant,
    sendMessage,
    updateMessage,
    deleteMessage,
    restoreMessage,
    removeFromQuote,
    forwardMessage,
    addReaction,
    removeReaction,
    toggleFavorite,
    sendTyping,
    syncMessagesAfterReconnect,
    syncAllChatsAfterReconnect,
    cleanup,
  }
})
