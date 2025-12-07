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
} from '@/types'

const API_BASE = '/api'

class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
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
    requireAuth = true
  ): Promise<T> {
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

  // Typing indicator
  async sendTypingIndicator(chatId: string, isTyping: boolean): Promise<void> {
    return this.request<void>('POST', `/chats/${chatId}/typing`, { is_typing: isTyping })
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
}

export const api = new ApiClient()
export { ApiError }
