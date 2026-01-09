package handler

// Swagger model definitions for API documentation

// Auth Models

// RegisterRequest represents registration data
type RegisterRequest struct {
	Username string `json:"username" example:"johndoe"`
	Email    string `json:"email" example:"john@example.com"`
	Password string `json:"password" example:"secretpassword"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" example:"john@example.com"`
	Password string `json:"password" example:"secretpassword"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

// AuthResponse represents authentication response with tokens
type AuthResponse struct {
	AccessToken  string       `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string       `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn    int64        `json:"expires_in" example:"900"`
	User         UserResponse `json:"user"`
}

// UserResponse represents user information
type UserResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string `json:"username" example:"johndoe"`
	Email     string `json:"email" example:"john@example.com"`
	Role      string `json:"role" example:"user"`
	CreatedAt string `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt string `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// UpdateUserRequest represents user update data
type UpdateUserRequest struct {
	Username string `json:"username,omitempty" example:"newusername"`
	Email    string `json:"email,omitempty" example:"newemail@example.com"`
}

// ChangePasswordRequest represents password change data
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" example:"oldpassword"`
	NewPassword     string `json:"new_password" example:"newpassword"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// Centrifugo Models

// ConnectionTokenResponse represents Centrifugo connection token response
type ConnectionTokenResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresAt int64  `json:"expires_at" example:"1705320600"`
}

// SubscriptionTokenRequest represents subscription token request
type SubscriptionTokenRequest struct {
	Channel string `json:"channel" example:"chat:550e8400-e29b-41d4-a716-446655440000"`
}

// SubscriptionTokenResponse represents subscription token response
type SubscriptionTokenResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	Channel   string `json:"channel" example:"chat:550e8400-e29b-41d4-a716-446655440000"`
	ExpiresAt int64  `json:"expires_at" example:"1705320600"`
}

// Chat Models

// CreateChatRequest represents chat creation data
type CreateChatRequest struct {
	Name        string   `json:"name" example:"General Discussion"`
	Description string   `json:"description,omitempty" example:"A place for general chat"`
	Type        string   `json:"type" example:"group"`
	MemberIDs   []string `json:"member_ids,omitempty" example:"[\"user-id-1\",\"user-id-2\"]"`
}

// ChatResponse represents a chat
type ChatResponse struct {
	ID          string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string   `json:"name" example:"General Discussion"`
	Description string   `json:"description" example:"A place for general chat"`
	Type        string   `json:"type" example:"group"`
	CreatorID   string   `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt   string   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   string   `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	MemberIDs   []string `json:"member_ids,omitempty"`
}

// ChatListResponse represents a list of chats
type ChatListResponse struct {
	Chats      []ChatResponse `json:"chats"`
	TotalCount int            `json:"total_count" example:"10"`
}

// UpdateChatRequest represents chat update data
type UpdateChatRequest struct {
	Name        string `json:"name,omitempty" example:"Updated Chat Name"`
	Description string `json:"description,omitempty" example:"Updated description"`
}

// AddMembersRequest represents add members request
type AddMembersRequest struct {
	MemberIDs []string `json:"member_ids" example:"[\"user-id-1\",\"user-id-2\"]"`
}

// RemoveMemberRequest represents remove member request
type RemoveMemberRequest struct {
	MemberID string `json:"member_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// Message Models

// SendMessageRequest represents message send data
type SendMessageRequest struct {
	Content  string   `json:"content" example:"Hello, world!"`
	ReplyTo  []string `json:"reply_to,omitempty" example:"[\"msg-id-1\"]"`
	FileIDs  []string `json:"file_ids,omitempty" example:"[\"file-id-1\"]"`
	ThreadID string   `json:"thread_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// MessageResponse represents a message
type MessageResponse struct {
	ID              string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID          string                 `json:"chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID        string                 `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderUsername  string                 `json:"sender_username" example:"johndoe"`
	Content         string                 `json:"content" example:"Hello, world!"`
	Type            string                 `json:"type,omitempty" example:"text"`
	CreatedAt       string                 `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt       string                 `json:"updated_at" example:"2024-01-15T10:30:00Z"`
	ReplyTo         []ReplyToMessage       `json:"reply_to,omitempty"`
	Files           []FileAttachment       `json:"files,omitempty"`
	Reactions       map[string]int         `json:"reactions,omitempty"`
	ThreadID        string                 `json:"thread_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ForwardedFrom   *ForwardedFromResponse `json:"forwarded_from,omitempty"`
	IsSystem        bool                   `json:"is_system,omitempty" example:"false"`
}

// ReplyToMessage represents a reply reference
type ReplyToMessage struct {
	ID             string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID       string `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderUsername string `json:"sender_username" example:"johndoe"`
	Content        string `json:"content" example:"Original message content"`
	CreatedAt      string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ForwardedFromResponse represents forwarded message info
type ForwardedFromResponse struct {
	ChatID         string `json:"chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatName       string `json:"chat_name" example:"Original Chat"`
	MessageID      string `json:"message_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID       string `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderUsername string `json:"sender_username" example:"originaluser"`
}

// MessageListResponse represents a list of messages
type MessageListResponse struct {
	Messages   []MessageResponse `json:"messages"`
	TotalCount int               `json:"total_count" example:"100"`
	HasMore    bool              `json:"has_more" example:"true"`
}

// UpdateMessageRequest represents message update data
type UpdateMessageRequest struct {
	Content string `json:"content" example:"Updated message content"`
}

// AddReactionRequest represents reaction add request
type AddReactionRequest struct {
	Emoji string `json:"emoji" example:"👍"`
}

// ForwardMessageRequest represents forward message request
type ForwardMessageRequest struct {
	ToChatID string `json:"to_chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Comment  string `json:"comment,omitempty" example:"Check this out!"`
}

// Thread Models

// CreateThreadRequest represents thread creation data
type CreateThreadRequest struct {
	Name      string `json:"name" example:"Bug Discussion"`
	MessageID string `json:"message_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ThreadResponse represents a thread
type ThreadResponse struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID       string `json:"chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string `json:"name" example:"Bug Discussion"`
	CreatorID    string `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageCount int    `json:"message_count" example:"15"`
	CreatedAt    string `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    string `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ThreadListResponse represents a list of threads
type ThreadListResponse struct {
	Threads    []ThreadResponse `json:"threads"`
	TotalCount int              `json:"total_count" example:"5"`
}

// File Models

// FileAttachment represents a file attachment
type FileAttachment struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Filename  string `json:"filename" example:"document.pdf"`
	Size      int64  `json:"size" example:"102400"`
	MimeType  string `json:"mime_type" example:"application/pdf"`
	URL       string `json:"url" example:"/api/files/550e8400-e29b-41d4-a716-446655440000/download"`
	CreatedAt string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// FileUploadResponse represents file upload response
type FileUploadResponse struct {
	ID       string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Filename string `json:"filename" example:"document.pdf"`
	Size     int64  `json:"size" example:"102400"`
	MimeType string `json:"mime_type" example:"application/pdf"`
	URL      string `json:"url" example:"/api/files/550e8400-e29b-41d4-a716-446655440000/download"`
}

// FileListResponse represents a list of files
type FileListResponse struct {
	Files      []FileAttachment `json:"files"`
	TotalCount int              `json:"total_count" example:"25"`
}

// Presence Models

// PresenceStatusRequest represents status update request
type PresenceStatusRequest struct {
	Status string `json:"status" example:"online"`
}

// PresenceListResponse represents list of user presence statuses
// Note: PresenceResponse is defined in presence.go
type PresenceListResponse struct {
	Presences []PresenceResponse `json:"presences"`
}

// Poll Models

// CreatePollRequest represents poll creation data
type CreatePollRequest struct {
	Question  string   `json:"question" example:"What should we build next?"`
	Options   []string `json:"options" example:"[\"Feature A\",\"Feature B\",\"Feature C\"]"`
	MultiVote bool     `json:"multi_vote" example:"false"`
	Anonymous bool     `json:"anonymous" example:"false"`
	ExpiresAt string   `json:"expires_at,omitempty" example:"2024-01-20T10:30:00Z"`
}

// PollResponse represents a poll
type PollResponse struct {
	ID        string             `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MessageID string             `json:"message_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Question  string             `json:"question" example:"What should we build next?"`
	Options   []PollOptionResponse `json:"options"`
	MultiVote bool               `json:"multi_vote" example:"false"`
	Anonymous bool               `json:"anonymous" example:"false"`
	CreatorID string             `json:"creator_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExpiresAt string             `json:"expires_at,omitempty" example:"2024-01-20T10:30:00Z"`
	CreatedAt string             `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// PollOptionResponse represents a poll option
type PollOptionResponse struct {
	ID         string   `json:"id" example:"1"`
	Text       string   `json:"text" example:"Feature A"`
	VoteCount  int      `json:"vote_count" example:"5"`
	VoterIDs   []string `json:"voter_ids,omitempty"`
	Percentage float64  `json:"percentage" example:"50.0"`
}

// VotePollRequest represents poll vote request
type VotePollRequest struct {
	OptionIDs []string `json:"option_ids" example:"[\"1\"]"`
}

// Common Models

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Limit  int `json:"limit" example:"20"`
	Offset int `json:"offset" example:"0"`
}

// Voice Models

// CreateConferenceRequest represents conference creation data
type CreateConferenceRequest struct {
	Name            string `json:"name" example:"Team Standup"`
	ChatID          string `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	MaxMembers      int    `json:"max_members,omitempty" example:"10"`
	IsPrivate       bool   `json:"is_private" example:"false"`
	EnableRecording bool   `json:"enable_recording" example:"true"`
}

// ConferenceResponse represents a conference
type ConferenceResponse struct {
	ID               string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string `json:"name" example:"Team Standup"`
	FreeSwitchName   string `json:"freeswitch_name" example:"adhoc_chat_abc123"`
	ChatID           string `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedBy        string `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status           string `json:"status" example:"active"`
	MaxMembers       int    `json:"max_members" example:"10"`
	ParticipantCount int    `json:"participant_count" example:"3"`
	IsPrivate        bool   `json:"is_private" example:"false"`
	RecordingPath    string `json:"recording_path,omitempty" example:"/recordings/conf_abc123.wav"`
	StartedAt        string `json:"started_at,omitempty" example:"2024-01-15T10:30:00Z"`
	EndedAt          string `json:"ended_at,omitempty" example:"2024-01-15T11:30:00Z"`
	CreatedAt        string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ListConferencesResponse represents list of conferences
type ListConferencesResponse struct {
	Conferences []ConferenceResponse `json:"conferences"`
	Total       int                  `json:"total" example:"5"`
}

// JoinConferenceRequest represents join conference options
type JoinConferenceRequest struct {
	Muted       bool   `json:"muted" example:"false"`
	DisplayName string `json:"display_name" example:"John Doe"`
}

// ParticipantResponse represents a conference participant
type ParticipantResponse struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ConferenceID string `json:"conference_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status       string `json:"status" example:"joined"`
	IsMuted      bool   `json:"is_muted" example:"false"`
	IsDeaf       bool   `json:"is_deaf" example:"false"`
	IsSpeaking   bool   `json:"is_speaking" example:"true"`
	Username     string `json:"username,omitempty" example:"johndoe"`
	DisplayName  string `json:"display_name,omitempty" example:"John Doe"`
	AvatarURL    string `json:"avatar_url,omitempty" example:"/avatars/johndoe.png"`
	JoinedAt     string `json:"joined_at,omitempty" example:"2024-01-15T10:30:00Z"`
}

// ParticipantsResponse represents list of participants
type ParticipantsResponse struct {
	Participants []ParticipantResponse `json:"participants"`
}

// MuteParticipantRequest represents mute/unmute request
type MuteParticipantRequest struct {
	Mute bool `json:"mute" example:"true"`
}

// InitiateCallRequest represents call initiation data
type InitiateCallRequest struct {
	CalleeID string `json:"callee_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID   string `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// CallResponse represents a call
type CallResponse struct {
	ID                string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CallerID          string `json:"caller_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CalleeID          string `json:"callee_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID            string `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ConferenceID      string `json:"conference_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status            string `json:"status" example:"answered"`
	Duration          int    `json:"duration" example:"180"`
	CallerUsername    string `json:"caller_username,omitempty" example:"johndoe"`
	CallerDisplayName string `json:"caller_display_name,omitempty" example:"John Doe"`
	CalleeUsername    string `json:"callee_username,omitempty" example:"janedoe"`
	CalleeDisplayName string `json:"callee_display_name,omitempty" example:"Jane Doe"`
	StartedAt         string `json:"started_at,omitempty" example:"2024-01-15T10:30:00Z"`
	AnsweredAt        string `json:"answered_at,omitempty" example:"2024-01-15T10:30:05Z"`
	EndedAt           string `json:"ended_at,omitempty" example:"2024-01-15T10:33:05Z"`
}

// CallHistoryResponse represents call history list
type CallHistoryResponse struct {
	Calls []CallResponse `json:"calls"`
	Total int            `json:"total" example:"25"`
}

// IceServerResponse represents an ICE server configuration
type IceServerResponse struct {
	URLs       []string `json:"urls" example:"[\"turn:turn.example.com:3478\"]"`
	Username   string   `json:"username,omitempty" example:"turnuser"`
	Credential string   `json:"credential,omitempty" example:"turnsecret"`
}

// VertoCredentialsResponse represents Verto connection credentials
type VertoCredentialsResponse struct {
	UserID     string              `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Login      string              `json:"login" example:"user_abc123@chatapp.local"`
	Password   string              `json:"password" example:"randomsecret123"`
	WSUrl      string              `json:"ws_url" example:"wss://chatapp.local:8081"`
	IceServers []IceServerResponse `json:"ice_servers"`
	ExpiresAt  int64               `json:"expires_at" example:"1705320600"`
}

// ChatCallResponse represents response when starting call from chat
type ChatCallResponse struct {
	Conference  ConferenceResponse       `json:"conference"`
	Credentials VertoCredentialsResponse `json:"credentials"`
}

// StartChatCallRequest represents request to start call from chat
type StartChatCallRequest struct {
	Name string `json:"name,omitempty" example:"Team Meeting"`
}

// Scheduled Events Models

// RecurrenceRuleRequest represents recurrence rule for scheduled events
type RecurrenceRuleRequest struct {
	Frequency  string  `json:"frequency" example:"WEEKLY"`
	DaysOfWeek []int32 `json:"days_of_week,omitempty"`
	DayOfMonth int     `json:"day_of_month,omitempty" example:"15"`
	Until      string  `json:"until,omitempty" example:"2024-12-31T23:59:59Z"`
	Count      int     `json:"count,omitempty" example:"10"`
}

// ScheduleConferenceRequest represents scheduled conference creation
type ScheduleConferenceRequest struct {
	Name            string                 `json:"name" example:"Weekly Standup"`
	ChatID          string                 `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ScheduledAt     string                 `json:"scheduled_at" example:"2024-01-20T10:00:00Z"`
	Recurrence      *RecurrenceRuleRequest `json:"recurrence,omitempty"`
	ParticipantIDs  []string               `json:"participant_ids,omitempty" example:"[\"user-id-1\",\"user-id-2\"]"`
	MaxMembers      int                    `json:"max_members,omitempty" example:"50"`
	EnableRecording bool                   `json:"enable_recording" example:"true"`
}

// CreateAdHocFromChatRequest represents ad-hoc call from chat
type CreateAdHocFromChatRequest struct {
	ChatID         string   `json:"chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParticipantIDs []string `json:"participant_ids,omitempty" example:"[\"user-id-1\",\"user-id-2\"]"`
}

// CreateQuickAdHocRequest represents quick ad-hoc call
type CreateQuickAdHocRequest struct {
	Name string `json:"name,omitempty" example:"Quick Call"`
}

// UpdateRSVPRequest represents RSVP update
type UpdateRSVPRequest struct {
	Status string `json:"status" example:"accepted"`
}

// UpdateParticipantRoleRequest represents role update
type UpdateParticipantRoleRequest struct {
	Role string `json:"role" example:"moderator"`
}

// AddParticipantsRequest represents add participants request
type AddParticipantsRequest struct {
	UserIDs     []string `json:"user_ids" example:"[\"user-id-1\",\"user-id-2\"]"`
	DefaultRole string   `json:"default_role,omitempty" example:"participant"`
}

// RecurrenceRuleResponse represents recurrence rule response
type RecurrenceRuleResponse struct {
	Frequency  string  `json:"frequency" example:"weekly"`
	DaysOfWeek []int32 `json:"days_of_week,omitempty"`
	DayOfMonth int     `json:"day_of_month,omitempty" example:"15"`
	Until      string  `json:"until,omitempty" example:"2024-12-31T23:59:59Z"`
	Count      int     `json:"count,omitempty" example:"10"`
}

// ScheduledConferenceResponse represents a scheduled conference
type ScheduledConferenceResponse struct {
	ID               string                        `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string                        `json:"name" example:"Weekly Standup"`
	FreeSwitchName   string                        `json:"freeswitch_name" example:"adhoc_chat_abc123"`
	ChatID           string                        `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedBy        string                        `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status           string                        `json:"status" example:"scheduled"`
	EventType        string                        `json:"event_type" example:"scheduled"`
	ScheduledAt      string                        `json:"scheduled_at,omitempty" example:"2024-01-20T10:00:00Z"`
	SeriesID         string                        `json:"series_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	MaxMembers       int                           `json:"max_members" example:"50"`
	ParticipantCount int                           `json:"participant_count" example:"5"`
	AcceptedCount    int                           `json:"accepted_count" example:"3"`
	DeclinedCount    int                           `json:"declined_count" example:"1"`
	Recurrence       *RecurrenceRuleResponse       `json:"recurrence,omitempty"`
	Participants     []ScheduledParticipantResponse `json:"participants,omitempty"`
	CreatedAt        string                        `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ScheduledParticipantResponse represents a participant with role and RSVP
type ScheduledParticipantResponse struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ConferenceID string `json:"conference_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status       string `json:"status" example:"connecting"`
	Role         string `json:"role" example:"participant"`
	RSVPStatus   string `json:"rsvp_status" example:"accepted"`
	RSVPAt       string `json:"rsvp_at,omitempty" example:"2024-01-16T09:00:00Z"`
	Username     string `json:"username,omitempty" example:"johndoe"`
	DisplayName  string `json:"display_name,omitempty" example:"John Doe"`
	AvatarURL    string `json:"avatar_url,omitempty" example:"/avatars/johndoe.png"`
	JoinedAt     string `json:"joined_at,omitempty" example:"2024-01-15T10:30:00Z"`
}

// ListScheduledConferencesResponse represents list of scheduled conferences
type ListScheduledConferencesResponse struct {
	Conferences []ScheduledConferenceResponse `json:"conferences"`
	Total       int                           `json:"total" example:"10"`
}

// ChatConferencesResponse represents conferences for a chat
type ChatConferencesResponse struct {
	Conferences []ScheduledConferenceResponse `json:"conferences"`
}

// Conference History Models

// ConferenceHistoryResponse represents conference history with participants
type ConferenceHistoryResponse struct {
	ID               string                      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string                      `json:"name" example:"Team Standup"`
	ChatID           string                      `json:"chat_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Status           string                      `json:"status" example:"ended"`
	StartedAt        string                      `json:"started_at,omitempty" example:"2024-01-15T10:30:00Z"`
	EndedAt          string                      `json:"ended_at,omitempty" example:"2024-01-15T11:30:00Z"`
	CreatedAt        string                      `json:"created_at" example:"2024-01-15T10:30:00Z"`
	ParticipantCount int                         `json:"participant_count" example:"5"`
	ThreadID         string                      `json:"thread_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AllParticipants  []ParticipantHistoryResponse `json:"all_participants,omitempty"`
}

// ListConferenceHistoryResponse represents list of conference history
type ListConferenceHistoryResponse struct {
	Conferences []ConferenceHistoryResponse `json:"conferences"`
	Total       int                         `json:"total" example:"10"`
}

// ParticipantHistoryResponse represents participant with all sessions
type ParticipantHistoryResponse struct {
	UserID      string                     `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username    string                     `json:"username,omitempty" example:"johndoe"`
	DisplayName string                     `json:"display_name,omitempty" example:"John Doe"`
	Sessions    []ParticipantSessionResponse `json:"sessions"`
}

// ParticipantSessionResponse represents a single join/leave session
type ParticipantSessionResponse struct {
	JoinedAt string `json:"joined_at" example:"2024-01-15T10:30:00Z"`
	LeftAt   string `json:"left_at,omitempty" example:"2024-01-15T11:00:00Z"`
	Status   string `json:"status" example:"left"`
	Role     string `json:"role" example:"participant"`
}

// ConferenceMessageResponse represents a message during a conference
type ConferenceMessageResponse struct {
	ID                string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChatID            string `json:"chat_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderID          string `json:"sender_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SenderUsername    string `json:"sender_username,omitempty" example:"johndoe"`
	SenderDisplayName string `json:"sender_display_name,omitempty" example:"John Doe"`
	Content           string `json:"content" example:"Hello everyone!"`
	CreatedAt         string `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ConferenceMessagesResponse represents messages during a conference
type ConferenceMessagesResponse struct {
	Messages []ConferenceMessageResponse `json:"messages"`
}

// ModeratorActionResponse represents a moderator action
type ModeratorActionResponse struct {
	ID                string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ConferenceID      string `json:"conference_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ActorID           string `json:"actor_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TargetUserID      string `json:"target_user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	ActionType        string `json:"action_type" example:"mute"`
	Details           string `json:"details,omitempty" example:"{}"`
	ActorUsername     string `json:"actor_username,omitempty" example:"moderator"`
	ActorDisplayName  string `json:"actor_display_name,omitempty" example:"Moderator User"`
	TargetUsername    string `json:"target_username,omitempty" example:"participant"`
	TargetDisplayName string `json:"target_display_name,omitempty" example:"Participant User"`
	CreatedAt         string `json:"created_at" example:"2024-01-15T10:35:00Z"`
}

// ModeratorActionsResponse represents list of moderator actions
type ModeratorActionsResponse struct {
	Actions []ModeratorActionResponse `json:"actions"`
}

// Chat Files Models

// ChatFileResponse represents a file in a chat
type ChatFileResponse struct {
	ID           string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	LinkID       string `json:"link_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Filename     string `json:"filename" example:"document.pdf"`
	Size         int64  `json:"size" example:"102400"`
	MimeType     string `json:"mime_type,omitempty" example:"application/pdf"`
	UploadedBy   string `json:"uploaded_by" example:"550e8400-e29b-41d4-a716-446655440000"`
	UploaderName string `json:"uploader_name,omitempty" example:"John Doe"`
	UploadedAt   string `json:"uploaded_at" example:"2024-01-15T10:30:00Z"`
}

// ChatFilesResponse represents list of files in a chat
type ChatFilesResponse struct {
	Files []ChatFileResponse `json:"files"`
	Total int                `json:"total" example:"25"`
}
