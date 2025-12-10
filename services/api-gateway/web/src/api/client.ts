import type {
  User,
  AuthTokens,
  LoginRequest,
  RegisterRequest,
  Chat,
  Message,
  Participant,
  CreateChatRequest,
  SendMessageRequest,
  PresenceInfo,
  Thread,
  ThreadParticipant,
  CreateThreadRequest,
} from '@/types'

const API_BASE = '/api'

// Retry configuration
const RETRY_CONFIG = {
  maxRetries: 3,
  baseDelayMs: 1000,
  maxDelayMs: 10000,
  // HTTP codes that should trigger retry
  retryableStatusCodes: [408, 429, 500, 502, 503, 504],
}

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public isNetworkError: boolean = false
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

// Check if error is retryable
function isRetryableError(error: unknown): boolean {
  if (error instanceof ApiError) {
    return RETRY_CONFIG.retryableStatusCodes.includes(error.status) || error.isNetworkError
  }
  // Network errors (fetch failed)
  if (error instanceof TypeError && error.message.includes('fetch')) {
    return true
  }
  return false
}

// Calculate delay with exponential backoff + jitter
function getRetryDelay(attempt: number): number {
  const exponentialDelay = RETRY_CONFIG.baseDelayMs * Math.pow(2, attempt)
  const jitter = Math.random() * 0.3 * exponentialDelay // 0-30% jitter
  return Math.min(exponentialDelay + jitter, RETRY_CONFIG.maxDelayMs)
}

// Sleep helper
function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

class ApiClient {
  private accessToken: string | null = null

  constructor() {
    this.accessToken = localStorage.getItem('access_token')
  }

  setAccessToken(token: string | null) {
    this.accessToken = token
    if (token) {
      localStorage.setItem('access_token', token)
    } else {
      localStorage.removeItem('access_token')
    }
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    requireAuth = true,
    options: { retry?: boolean; maxRetries?: number } = {}
  ): Promise<T> {
    const { retry = true, maxRetries = RETRY_CONFIG.maxRetries } = options
    let lastError: Error | null = null

    for (let attempt = 0; attempt <= (retry ? maxRetries : 0); attempt++) {
      try {
        const headers: Record<string, string> = {
          'Content-Type': 'application/json',
        }

        if (requireAuth && this.accessToken) {
          headers['Authorization'] = `Bearer ${this.accessToken}`
        }

        const response = await fetch(`${API_BASE}${path}`, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
        })

        if (!response.ok) {
          const error = await response.json().catch(() => ({ error: 'Unknown error' }))
          throw new ApiError(response.status, error.message || error.error || 'Request failed')
        }

        if (response.status === 204) {
          return undefined as T
        }

        return response.json()

      } catch (error) {
        lastError = error instanceof Error ? error : new Error(String(error))

        // Check if it's a network error
        if (error instanceof TypeError && error.message.includes('fetch')) {
          lastError = new ApiError(0, 'Network error: Unable to connect', true)
        }

        // Should we retry?
        if (retry && attempt < maxRetries && isRetryableError(lastError)) {
          const delay = getRetryDelay(attempt)
          console.log(`[API] Request failed, retrying in ${delay}ms (attempt ${attempt + 1}/${maxRetries})`, path)
          await sleep(delay)
          continue
        }

        // No more retries
        throw lastError
      }
    }

    // Should never reach here, but just in case
    throw lastError || new Error('Unknown error')
  }

  // Request without retry (for fire-and-forget operations like typing indicator)
  private async requestNoRetry<T>(
    method: string,
    path: string,
    body?: unknown,
    requireAuth = true
  ): Promise<T> {
    return this.request<T>(method, path, body, requireAuth, { retry: false })
  }

  // Auth endpoints
  async register(data: RegisterRequest): Promise<AuthTokens> {
    const result = await this.request<AuthTokens>('POST', '/auth/register', data, false)
    this.setAccessToken(result.access_token)
    localStorage.setItem('refresh_token', result.refresh_token)
    return result
  }

  async login(data: LoginRequest): Promise<AuthTokens> {
    const result = await this.request<AuthTokens>('POST', '/auth/login', data, false)
    this.setAccessToken(result.access_token)
    localStorage.setItem('refresh_token', result.refresh_token)
    return result
  }

  async logout(): Promise<void> {
    const refreshToken = localStorage.getItem('refresh_token')
    await this.request<void>('POST', '/auth/logout', { refresh_token: refreshToken })
    this.setAccessToken(null)
    localStorage.removeItem('refresh_token')
  }

  async refreshToken(): Promise<AuthTokens> {
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) {
      throw new ApiError(401, 'No refresh token')
    }
    const result = await this.request<AuthTokens>(
      'POST',
      '/auth/refresh',
      { refresh_token: refreshToken },
      false
    )
    this.setAccessToken(result.access_token)
    localStorage.setItem('refresh_token', result.refresh_token)
    return result
  }

  async getCurrentUser(): Promise<User> {
    return this.request<User>('GET', '/auth/me')
  }

  async updateCurrentUser(data: Partial<User>): Promise<User> {
    return this.request<User>('PUT', '/auth/me', data)
  }

  // Chat endpoints
  async getChats(limit = 20, offset = 0): Promise<{ chats: Chat[]; total: number }> {
    return this.request<{ chats: Chat[]; total: number }>(
      'GET',
      `/chats?limit=${limit}&offset=${offset}`
    )
  }

  async getChat(chatId: string): Promise<Chat> {
    return this.request<Chat>('GET', `/chats/${chatId}`)
  }

  async createChat(data: CreateChatRequest): Promise<Chat> {
    return this.request<Chat>('POST', '/chats', data)
  }

  async updateChat(chatId: string, data: { name?: string; description?: string }): Promise<Chat> {
    return this.request<Chat>('PUT', `/chats/${chatId}`, data)
  }

  async deleteChat(chatId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/${chatId}`)
  }

  // Participants
  async getParticipants(chatId: string): Promise<{ participants: Participant[] }> {
    return this.request<{ participants: Participant[] }>('GET', `/chats/${chatId}/participants`)
  }

  async addParticipant(chatId: string, userId: string, role = 'member'): Promise<void> {
    return this.request<void>('POST', `/chats/${chatId}/participants`, { user_id: userId, role })
  }

  async removeParticipant(chatId: string, userId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/${chatId}/participants/${userId}`)
  }

  // Messages
  async getMessages(
    chatId: string,
    limit = 50,
    offset = 0,
    threadId?: string
  ): Promise<{ messages: Message[]; total: number }> {
    let url = `/chats/${chatId}/messages?limit=${limit}&offset=${offset}`
    if (threadId) {
      url += `&thread_id=${threadId}`
    }
    return this.request<{ messages: Message[]; total: number }>('GET', url)
  }

  async sendMessage(chatId: string, data: SendMessageRequest): Promise<Message> {
    return this.request<Message>('POST', `/chats/${chatId}/messages`, data)
  }

  async updateMessage(messageId: string, content: string): Promise<Message> {
    return this.request<Message>('PUT', `/chats/messages/${messageId}`, { content })
  }

  async deleteMessage(messageId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/messages/${messageId}`)
  }

  // Sync messages after reconnect
  async syncMessages(
    chatId: string,
    afterSeqNum: number,
    limit = 100
  ): Promise<{ messages: Message[]; has_more: boolean }> {
    return this.request<{ messages: Message[]; has_more: boolean }>(
      'GET',
      `/chats/${chatId}/messages/sync?after_seq=${afterSeqNum}&limit=${limit}`
    )
  }

  // Reactions
  async addReaction(messageId: string, emoji: string): Promise<void> {
    return this.request<void>('POST', `/chats/messages/${messageId}/reactions`, { emoji })
  }

  async removeReaction(messageId: string, emoji: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/messages/${messageId}/reactions/${emoji}`)
  }

  // Favorites & Archive
  async addToFavorites(chatId: string): Promise<void> {
    return this.request<void>('POST', `/chats/${chatId}/favorite`)
  }

  async removeFromFavorites(chatId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/${chatId}/favorite`)
  }

  async archiveChat(chatId: string): Promise<void> {
    return this.request<void>('POST', `/chats/${chatId}/archive`)
  }

  async unarchiveChat(chatId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/${chatId}/archive`)
  }

  // Typing indicator (no retry - fire and forget)
  async sendTypingIndicator(chatId: string, isTyping: boolean): Promise<void> {
    return this.requestNoRetry<void>('POST', `/chats/${chatId}/typing`, { is_typing: isTyping })
  }

  // Centrifugo
  async getCentrifugoConnectionToken(): Promise<{ token: string; expires_at: number }> {
    return this.request<{ token: string; expires_at: number }>('GET', '/centrifugo/connection-token')
  }

  async getCentrifugoSubscriptionToken(
    channel: string
  ): Promise<{ token: string; channel: string; expires_at: number }> {
    return this.request<{ token: string; channel: string; expires_at: number }>(
      'POST',
      '/centrifugo/subscription-token',
      { channel }
    )
  }

  // Presence
  async setPresenceStatus(status: string): Promise<PresenceInfo> {
    return this.request<PresenceInfo>('PUT', '/presence/status', { status })
  }

  async getMyPresence(): Promise<PresenceInfo> {
    return this.request<PresenceInfo>('GET', '/presence/status')
  }

  async getUsersPresence(userIds: string[]): Promise<{ presences: PresenceInfo[] }> {
    const ids = userIds.join(',')
    return this.request<{ presences: PresenceInfo[] }>('GET', `/presence/users?ids=${ids}`)
  }

  async registerConnection(connectionId: string): Promise<PresenceInfo> {
    return this.request<PresenceInfo>('POST', '/presence/connect', { connection_id: connectionId })
  }

  async unregisterConnection(connectionId: string): Promise<PresenceInfo> {
    return this.request<PresenceInfo>('POST', '/presence/disconnect', { connection_id: connectionId })
  }

  // File uploads
  async uploadFile(file: File): Promise<{ id: string; link_id: string; filename: string; original_filename: string; content_type: string; size: number }> {
    const formData = new FormData()
    formData.append('file', file)

    const headers: Record<string, string> = {}
    if (this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`
    }

    const response = await fetch('/api/files/upload', {
      method: 'POST',
      headers,
      body: formData,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }))
      throw new ApiError(response.status, error.message || error.error || 'Upload failed')
    }

    return response.json()
  }

  // Thread operations
  async listThreads(
    chatId: string,
    page = 1,
    count = 20
  ): Promise<{ threads: Thread[]; total: number }> {
    return this.request<{ threads: Thread[]; total: number }>(
      'GET',
      `/chats/${chatId}/threads?page=${page}&count=${count}`
    )
  }

  async createThread(chatId: string, data: CreateThreadRequest): Promise<Thread> {
    return this.request<Thread>('POST', `/chats/${chatId}/threads`, data)
  }

  async getThread(threadId: string): Promise<Thread> {
    return this.request<Thread>('GET', `/chats/threads/${threadId}`)
  }

  async archiveThread(threadId: string): Promise<Thread> {
    return this.request<Thread>('POST', `/chats/threads/${threadId}/archive`)
  }

  async getThreadMessages(
    threadId: string,
    page = 1,
    count = 50
  ): Promise<{ messages: Message[]; total: number }> {
    return this.request<{ messages: Message[]; total: number }>(
      'GET',
      `/chats/threads/${threadId}/messages?page=${page}&count=${count}`
    )
  }

  // Create thread from message (reply thread)
  async createThreadFromMessage(messageId: string): Promise<Thread> {
    return this.request<Thread>('POST', `/chats/messages/${messageId}/thread`)
  }

  // Thread participant operations
  async addThreadParticipant(threadId: string, userId: string): Promise<void> {
    return this.request<void>('POST', `/chats/threads/${threadId}/participants`, { user_id: userId })
  }

  async removeThreadParticipant(threadId: string, userId: string): Promise<void> {
    return this.request<void>('DELETE', `/chats/threads/${threadId}/participants/${userId}`)
  }

  async listThreadParticipants(threadId: string): Promise<{ participants: ThreadParticipant[] }> {
    return this.request<{ participants: ThreadParticipant[] }>(
      'GET',
      `/chats/threads/${threadId}/participants`
    )
  }

  // Subthread operations
  async listSubthreads(
    parentThreadId: string,
    page = 1,
    count = 20
  ): Promise<{ threads: Thread[]; total: number }> {
    return this.request<{ threads: Thread[]; total: number }>(
      'GET',
      `/chats/threads/${parentThreadId}/subthreads?page=${page}&count=${count}`
    )
  }

  async createSubthread(parentThreadId: string, data: CreateThreadRequest): Promise<Thread> {
    return this.request<Thread>('POST', `/chats/threads/${parentThreadId}/subthreads`, data)
  }
}

export const api = new ApiClient()
export { ApiError }
