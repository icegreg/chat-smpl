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
  Conference,
  VoiceParticipant,
  Call,
  VertoCredentials,
  CreateConferenceRequest,
  JoinConferenceRequest,
  InitiateCallRequest,
  StartChatCallResponse,
  ScheduledConference,
  ConferenceParticipant,
  ScheduleConferenceRequest,
  CreateAdHocFromChatRequest,
  RSVPStatus,
  ConferenceRole,
  ConferenceHistory,
  ModeratorAction,
  ChatFile,
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
  private isRefreshing = false
  private refreshPromise: Promise<AuthTokens> | null = null
  private onAuthFailure: (() => void) | null = null

  constructor() {
    this.accessToken = localStorage.getItem('access_token')
  }

  // Set callback for auth failure (logout + redirect)
  setOnAuthFailure(callback: () => void) {
    this.onAuthFailure = callback
  }

  setAccessToken(token: string | null) {
    this.accessToken = token
    if (token) {
      localStorage.setItem('access_token', token)
    } else {
      localStorage.removeItem('access_token')
    }
  }

  // Clear all auth data
  clearAuth() {
    this.accessToken = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  // Handle auth failure - clear tokens and trigger callback
  private handleAuthFailure() {
    this.clearAuth()
    if (this.onAuthFailure) {
      this.onAuthFailure()
    }
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    requireAuth = true,
    options: { retry?: boolean; maxRetries?: number; skipAuthRefresh?: boolean } = {}
  ): Promise<T> {
    const { retry = true, maxRetries = RETRY_CONFIG.maxRetries, skipAuthRefresh = false } = options
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

        // Handle 401 - try to refresh token
        if (response.status === 401 && requireAuth && !skipAuthRefresh) {
          const refreshed = await this.tryRefreshToken()
          if (refreshed) {
            // Retry the request with new token
            return this.request<T>(method, path, body, requireAuth, { ...options, skipAuthRefresh: true })
          } else {
            // Refresh failed - trigger auth failure
            this.handleAuthFailure()
            throw new ApiError(401, 'Session expired. Please login again.')
          }
        }

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

        // Don't retry 401 errors
        if (lastError instanceof ApiError && lastError.status === 401) {
          throw lastError
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

  // Try to refresh token, returns true if successful
  private async tryRefreshToken(): Promise<boolean> {
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) {
      return false
    }

    // Prevent multiple simultaneous refresh attempts
    if (this.isRefreshing) {
      try {
        await this.refreshPromise
        return true
      } catch {
        return false
      }
    }

    this.isRefreshing = true
    this.refreshPromise = this.doRefreshToken(refreshToken)

    try {
      await this.refreshPromise
      return true
    } catch (error) {
      console.error('[API] Token refresh failed:', error)
      return false
    } finally {
      this.isRefreshing = false
      this.refreshPromise = null
    }
  }

  private async doRefreshToken(refreshToken: string): Promise<AuthTokens> {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })

    if (!response.ok) {
      throw new ApiError(response.status, 'Token refresh failed')
    }

    const result = await response.json() as AuthTokens
    this.setAccessToken(result.access_token)
    localStorage.setItem('refresh_token', result.refresh_token)
    return result
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

  // Voice/Conference endpoints
  async createConference(data: CreateConferenceRequest): Promise<Conference> {
    return this.request<Conference>('POST', '/voice/conferences', data)
  }

  async getConference(conferenceId: string): Promise<Conference> {
    return this.request<Conference>('GET', `/voice/conferences/${conferenceId}`)
  }

  async getConferenceByFSName(fsName: string): Promise<Conference> {
    return this.request<Conference>('GET', `/voice/conferences/by-fs-name/${fsName}`)
  }

  async listConferences(chatId?: string, status?: string): Promise<{ conferences: Conference[] }> {
    let url = '/voice/conferences'
    const params: string[] = []
    if (chatId) params.push(`chat_id=${chatId}`)
    if (status) params.push(`status=${status}`)
    if (params.length > 0) url += '?' + params.join('&')
    return this.request<{ conferences: Conference[] }>('GET', url)
  }

  async getActiveConferences(): Promise<{ conferences: Conference[]; total: number }> {
    return this.request<{ conferences: Conference[]; total: number }>('GET', '/voice/conferences/active')
  }

  async joinConference(conferenceId: string, data?: JoinConferenceRequest): Promise<VoiceParticipant> {
    return this.request<VoiceParticipant>('POST', `/voice/conferences/${conferenceId}/join`, data || {})
  }

  async leaveConference(conferenceId: string): Promise<void> {
    return this.request<void>('POST', `/voice/conferences/${conferenceId}/leave`)
  }

  async getConferenceParticipants(conferenceId: string): Promise<{ participants: VoiceParticipant[] }> {
    return this.request<{ participants: VoiceParticipant[] }>('GET', `/voice/conferences/${conferenceId}/participants`)
  }

  async muteParticipant(conferenceId: string, userId: string, mute: boolean): Promise<VoiceParticipant> {
    return this.request<VoiceParticipant>('POST', `/voice/conferences/${conferenceId}/participants/${userId}/mute`, { mute })
  }

  async kickParticipant(conferenceId: string, userId: string): Promise<void> {
    return this.request<void>('DELETE', `/voice/conferences/${conferenceId}/participants/${userId}`)
  }

  async endConference(conferenceId: string): Promise<void> {
    return this.request<void>('DELETE', `/voice/conferences/${conferenceId}`)
  }

  // Call operations
  async initiateCall(data: InitiateCallRequest): Promise<Call> {
    return this.request<Call>('POST', '/voice/calls', data)
  }

  async answerCall(callId: string): Promise<Call> {
    return this.request<Call>('POST', `/voice/calls/${callId}/answer`)
  }

  async hangupCall(callId: string): Promise<void> {
    return this.request<void>('POST', `/voice/calls/${callId}/hangup`)
  }

  // Verto credentials
  async getVertoCredentials(): Promise<VertoCredentials> {
    return this.request<VertoCredentials>('GET', '/voice/credentials')
  }

  // Quick call from chat
  async startChatCall(chatId: string, chatName?: string): Promise<StartChatCallResponse> {
    return this.request<StartChatCallResponse>('POST', `/voice/chats/${chatId}/call`, { name: chatName })
  }

  // Scheduled Events API

  // Schedule a new conference (one-time or recurring)
  async scheduleConference(data: ScheduleConferenceRequest): Promise<ScheduledConference> {
    return this.request<ScheduledConference>('POST', '/voice/conferences/schedule', data)
  }

  // Create ad-hoc conference from chat
  async createAdHocFromChat(data: CreateAdHocFromChatRequest): Promise<ScheduledConference> {
    return this.request<ScheduledConference>('POST', '/voice/conferences/adhoc-chat', data)
  }

  // Create quick ad-hoc conference (without chat)
  async createQuickAdHoc(name?: string): Promise<ScheduledConference> {
    return this.request<ScheduledConference>('POST', '/voice/conferences/adhoc', { name })
  }

  // Update RSVP status for a conference
  async updateRSVP(conferenceId: string, status: RSVPStatus): Promise<ConferenceParticipant> {
    return this.request<ConferenceParticipant>('PUT', `/voice/conferences/${conferenceId}/rsvp`, { rsvp_status: status })
  }

  // Update participant role in a conference
  async updateParticipantRole(
    conferenceId: string,
    userId: string,
    newRole: ConferenceRole
  ): Promise<ConferenceParticipant> {
    return this.request<ConferenceParticipant>(
      'PUT',
      `/voice/conferences/${conferenceId}/participants/${userId}/role`,
      { new_role: newRole }
    )
  }

  // Add participants to a conference
  async addConferenceParticipants(
    conferenceId: string,
    userIds: string[],
    defaultRole?: ConferenceRole
  ): Promise<void> {
    return this.request<void>('POST', `/voice/conferences/${conferenceId}/participants`, {
      user_ids: userIds,
      default_role: defaultRole,
    })
  }

  // Remove participant from a conference
  async removeConferenceParticipant(conferenceId: string, userId: string): Promise<void> {
    return this.request<void>('DELETE', `/voice/conferences/${conferenceId}/participants/${userId}`)
  }

  // List scheduled conferences for current user
  async listScheduledConferences(
    upcomingOnly = true,
    limit = 50,
    offset = 0
  ): Promise<{ conferences: ScheduledConference[]; total: number }> {
    return this.request<{ conferences: ScheduledConference[]; total: number }>(
      'GET',
      `/voice/conferences/scheduled?upcoming_only=${upcomingOnly}&limit=${limit}&offset=${offset}`
    )
  }

  // Get conferences for a specific chat (for widget)
  async getChatConferences(
    chatId: string,
    upcomingOnly = true
  ): Promise<{ conferences: ScheduledConference[] }> {
    return this.request<{ conferences: ScheduledConference[] }>(
      'GET',
      `/voice/chats/${chatId}/conferences?upcoming_only=${upcomingOnly}`
    )
  }

  // Cancel a scheduled conference
  async cancelConference(conferenceId: string, cancelSeries = false): Promise<void> {
    return this.request<void>('DELETE', `/voice/conferences/${conferenceId}?cancel_series=${cancelSeries}`)
  }

  // Get scheduled conference with participants
  async getScheduledConference(conferenceId: string): Promise<ScheduledConference> {
    return this.request<ScheduledConference>('GET', `/voice/conferences/${conferenceId}/scheduled`)
  }

  // ======== Conference History ========

  // Get conference history for a chat
  async getChatConferenceHistory(
    chatId: string,
    limit = 20,
    offset = 0
  ): Promise<{ conferences: ConferenceHistory[]; total: number }> {
    return this.request<{ conferences: ConferenceHistory[]; total: number }>(
      'GET',
      `/voice/chats/${chatId}/conferences/history?limit=${limit}&offset=${offset}`
    )
  }

  // Get detailed conference history
  async getConferenceHistory(conferenceId: string): Promise<ConferenceHistory> {
    return this.request<ConferenceHistory>('GET', `/voice/conferences/${conferenceId}/history`)
  }

  // Get messages sent during a conference
  async getConferenceMessages(conferenceId: string): Promise<{ messages: Message[] }> {
    return this.request<{ messages: Message[] }>('GET', `/voice/conferences/${conferenceId}/messages`)
  }

  // Get moderator actions for a conference (moderator only)
  async getModeratorActions(conferenceId: string): Promise<{ actions: ModeratorAction[] }> {
    return this.request<{ actions: ModeratorAction[] }>(
      'GET',
      `/voice/conferences/${conferenceId}/moderator-actions`
    )
  }

  // ======== Chat Files ========

  // Get files in a chat
  async getChatFiles(
    chatId: string,
    limit = 50,
    offset = 0
  ): Promise<{ files: ChatFile[]; total: number }> {
    return this.request<{ files: ChatFile[]; total: number }>(
      'GET',
      `/files/chats/${chatId}/files?limit=${limit}&offset=${offset}`
    )
  }
}

export const api = new ApiClient()
export { ApiError }
