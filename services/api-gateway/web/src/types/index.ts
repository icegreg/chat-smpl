export interface User {
  id: string
  email: string
  username: string
  display_name: string
  role: 'owner' | 'moderator' | 'user' | 'guest'
  avatar_url?: string
  created_at: string
  updated_at: string
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  expires_at: number
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  username: string
  password: string
  display_name?: string
}

export interface Chat {
  id: string
  type: 'direct' | 'group' | 'channel'
  name: string
  description?: string
  created_by: string
  created_at: string
  updated_at: string
  participant_count: number
  last_message?: Message
  unread_count?: number
  is_favorite?: boolean
  is_archived?: boolean
}

export interface Participant {
  user_id: string
  chat_id: string
  role: 'owner' | 'admin' | 'member' | 'guest'
  joined_at: string
  username?: string
  email?: string
  display_name?: string
  avatar_url?: string
  user?: User
}

// Protobuf timestamp format
export interface ProtobufTimestamp {
  seconds: number
  nanos?: number
}

export interface Message {
  id: string
  chat_id: string
  sender_id: string
  content: string
  type: 'text' | 'file' | 'system'
  reply_to_id?: string
  thread_id?: string
  is_edited: boolean
  created_at: string
  updated_at: string
  sent_at?: string | ProtobufTimestamp
  sender?: User
  sender_username?: string
  sender_display_name?: string
  sender_avatar_url?: string
  reactions?: Reaction[]
  reply_to?: Message
  file_attachments?: FileAttachment[]
}

export interface Reaction {
  emoji: string
  count: number
  users: string[]
}

export interface FileAttachment {
  id: string
  link_id: string
  filename: string
  original_filename: string
  content_type: string
  size: number
}

export interface CreateChatRequest {
  type: 'direct' | 'group' | 'channel'
  name: string
  description?: string
  participant_ids: string[]
}

export interface SendMessageRequest {
  content: string
  reply_to_id?: string
  file_link_ids?: string[]
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
}

export interface CentrifugoEvent {
  type: string
  chat_id?: string
  user_id?: string
  data: unknown
  timestamp: string
}

// Presence types
export type UserStatus = 'available' | 'busy' | 'away' | 'dnd'

export interface PresenceInfo {
  user_id: string
  status: UserStatus
  is_online: boolean
  connection_count: number
  last_seen_at?: number
}
