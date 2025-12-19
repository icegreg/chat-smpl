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
  type?: 'text' | 'file' | 'system'
  reply_to_id?: string           // Deprecated: use reply_to_ids
  reply_to_ids?: string[]        // IDs of messages this is replying to
  thread_id?: string
  forwarded_from_id?: string // Original message ID if forwarded
  forwarded_from_chat_id?: string // Original chat ID if forwarded
  is_edited?: boolean
  is_forwarded?: boolean
  created_at: string
  updated_at?: string
  sent_at?: string | ProtobufTimestamp
  sender?: User
  sender_username?: string
  sender_display_name?: string
  sender_avatar_url?: string
  reactions?: Reaction[]
  reply_to?: Message             // Deprecated: use reply_to_messages
  reply_to_messages?: Message[]  // Full message data for replies
  forwarded_from?: ForwardedMessageInfo // Info about original forwarded message
  file_attachments?: FileAttachment[]
  seq_num?: number // Sequence number for reliable sync
  is_pending?: boolean // True if message is waiting to be sent (offline mode)
}

export interface ForwardedMessageInfo {
  original_message_id: string
  original_chat_id: string
  original_chat_name?: string
  original_sender_name?: string
  original_sent_at?: string
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
  reply_to_id?: string       // Deprecated: use reply_to_ids
  reply_to_ids?: string[]    // IDs of messages to reply to (supports multiple)
  thread_id?: string
  file_link_ids?: string[]
  forwarded_from_id?: string // Original message ID when forwarding
  forwarded_from_chat_id?: string // Original chat ID when forwarding
}

export interface ForwardMessageRequest {
  message_id: string // Message to forward
  source_chat_id: string // Original chat
  target_chat_id: string // Destination chat
  comment?: string // Optional comment with forwarded message
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

// Thread types
export type ThreadType = 'user' | 'system'

export interface Thread {
  id: string
  chat_id: string
  parent_message_id?: string
  parent_thread_id?: string  // For subthreads
  depth: number              // Nesting level (0 = top-level, 1 = subthread, etc.)
  thread_type: ThreadType
  title?: string
  message_count: number
  last_message_at?: string
  created_by?: string
  created_at: string
  updated_at: string
  is_archived: boolean
  restricted_participants: boolean
}

export interface ThreadParticipant {
  id: string
  thread_id: string
  user_id: string
  added_at: string
}

export interface CreateThreadRequest {
  parent_message_id?: string
  parent_thread_id?: string  // For creating subthreads
  thread_type?: ThreadType
  title?: string
  restricted_participants?: boolean
}

// Voice/Conference types
export type ConferenceStatus = 'active' | 'ended'
export type ParticipantStatus = 'connecting' | 'joined' | 'left' | 'kicked'
export type CallStatus = 'initiated' | 'ringing' | 'answered' | 'ended' | 'missed' | 'failed'

export interface Conference {
  id: string
  name: string
  chat_id?: string
  created_by: string
  status: ConferenceStatus
  max_members: number
  participant_count: number
  recording_path?: string
  started_at?: string
  created_at: string
}

export interface VoiceParticipant {
  id: string
  conference_id: string
  user_id: string
  status: ParticipantStatus
  is_muted: boolean
  is_speaking: boolean
  username?: string
  display_name?: string
  avatar_url?: string
  joined_at?: string
}

export interface Call {
  id: string
  caller_id: string
  callee_id: string
  chat_id?: string
  status: CallStatus
  duration: number
  caller_username?: string
  callee_username?: string
  started_at?: string
}

export interface IceServer {
  urls: string[]
  username?: string
  credential?: string
}

export interface VertoCredentials {
  user_id: string
  login: string
  password: string
  ws_url: string
  ice_servers: IceServer[]
  expires_at: number
}

export interface CreateConferenceRequest {
  name: string
  chat_id?: string
  max_members?: number
}

export interface JoinConferenceRequest {
  muted?: boolean
}

export interface InitiateCallRequest {
  callee_id: string
  chat_id?: string
}

export interface StartChatCallResponse {
  conference: Conference
  credentials: VertoCredentials
}

// Voice WebSocket events
export interface VoiceEvent {
  type: string
  data: unknown
}

export interface ConferenceEvent {
  id: string
  name: string
  chat_id?: string
  created_by: string
  status: ConferenceStatus
  participant_count: number
}

export interface ParticipantEvent {
  id: string
  conference_id: string
  user_id: string
  status: ParticipantStatus
  is_muted: boolean
  is_speaking: boolean
  username?: string
  display_name?: string
}

export interface CallEvent {
  id: string
  caller_id: string
  callee_id: string
  status: CallStatus
  caller_username?: string
  callee_username?: string
  caller_display_name?: string
  callee_display_name?: string
}

// Scheduled Events types
export type EventType = 'adhoc' | 'adhoc_chat' | 'scheduled' | 'recurring'
export type ConferenceRole = 'originator' | 'moderator' | 'speaker' | 'assistant' | 'participant'
export type RSVPStatus = 'pending' | 'accepted' | 'declined'
export type RecurrenceFrequency = 'daily' | 'weekly' | 'biweekly' | 'monthly'

export interface RecurrenceRule {
  frequency: RecurrenceFrequency
  days_of_week?: number[]
  day_of_month?: number
  until?: string
  count?: number
}

export interface ScheduledConference extends Conference {
  event_type: EventType
  scheduled_at?: string
  series_id?: string
  accepted_count: number
  declined_count: number
  recurrence?: RecurrenceRule
  participants?: ConferenceParticipant[]
}

export interface ConferenceParticipant extends VoiceParticipant {
  role: ConferenceRole
  rsvp_status: RSVPStatus
  rsvp_at?: string
}

export interface ScheduleConferenceRequest {
  name: string
  chat_id?: string
  scheduled_at: string
  recurrence?: RecurrenceRule
  participant_user_ids?: string[]
  max_members?: number
}

export interface CreateAdHocFromChatRequest {
  chat_id: string
  participant_user_ids?: string[]
}

export interface UpdateRSVPRequest {
  rsvp_status: RSVPStatus
}

export interface UpdateParticipantRoleRequest {
  new_role: ConferenceRole
}

export interface AddParticipantsRequest {
  user_ids: string[]
  default_role?: ConferenceRole
}

// Scheduled Events WebSocket events
export interface ScheduledConferenceEvent extends ConferenceEvent {
  event_type: EventType
  scheduled_at?: string
  accepted_count: number
  declined_count: number
}

export interface RSVPUpdatedEvent {
  conference_id: string
  user_id: string
  rsvp_status: RSVPStatus
}

export interface ParticipantRoleChangedEvent {
  conference_id: string
  user_id: string
  old_role: ConferenceRole
  new_role: ConferenceRole
}

export interface ConferenceReminderEvent {
  conference_id: string
  user_id: string
  conference_name: string
  scheduled_at: string
  minutes_before: number
}
