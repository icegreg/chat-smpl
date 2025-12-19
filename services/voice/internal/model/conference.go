package model

import (
	"time"

	"github.com/google/uuid"
)

type ConferenceStatus string

const (
	ConferenceStatusActive    ConferenceStatus = "active"
	ConferenceStatusEnded     ConferenceStatus = "ended"
	ConferenceStatusScheduled ConferenceStatus = "scheduled"
	ConferenceStatusCancelled ConferenceStatus = "cancelled"
)

// EventType represents the type of conference/meeting
type EventType string

const (
	EventTypeAdhoc     EventType = "adhoc"      // Quick call without chat
	EventTypeAdhocChat EventType = "adhoc_chat" // Ad-hoc from chat
	EventTypeScheduled EventType = "scheduled"  // One-time scheduled event
	EventTypeRecurring EventType = "recurring"  // Recurring event
)

// ConferenceRole represents a participant's role in the conference
type ConferenceRole string

const (
	RoleOriginator  ConferenceRole = "originator"  // Organizer - full control
	RoleModerator   ConferenceRole = "moderator"   // Moderator - can manage participants
	RoleSpeaker     ConferenceRole = "speaker"     // Speaker - logical role
	RoleAssistant   ConferenceRole = "assistant"   // Assistant - logical role
	RoleParticipant ConferenceRole = "participant" // Regular participant
)

// RSVPStatus represents participant's RSVP status
type RSVPStatus string

const (
	RSVPPending  RSVPStatus = "pending"
	RSVPAccepted RSVPStatus = "accepted"
	RSVPDeclined RSVPStatus = "declined"
)

// RecurrenceFrequency for recurring events
type RecurrenceFrequency string

const (
	RecurrenceDaily    RecurrenceFrequency = "daily"
	RecurrenceWeekly   RecurrenceFrequency = "weekly"
	RecurrenceBiweekly RecurrenceFrequency = "biweekly"
	RecurrenceMonthly  RecurrenceFrequency = "monthly"
)

// RecurrenceRule defines recurrence pattern for recurring events
type RecurrenceRule struct {
	ID              uuid.UUID           `json:"id" db:"id"`
	ConferenceID    uuid.UUID           `json:"conference_id" db:"conference_id"`
	Frequency       RecurrenceFrequency `json:"frequency" db:"frequency"`
	DaysOfWeek      []int               `json:"days_of_week" db:"days_of_week"`
	DayOfMonth      *int                `json:"day_of_month,omitempty" db:"day_of_month"`
	UntilDate       *time.Time          `json:"until_date,omitempty" db:"until_date"`
	OccurrenceCount *int                `json:"occurrence_count,omitempty" db:"occurrence_count"`
	CreatedAt       time.Time           `json:"created_at" db:"created_at"`
}

type Conference struct {
	ID             uuid.UUID        `json:"id" db:"id"`
	Name           string           `json:"name" db:"name"`
	ChatID         *uuid.UUID       `json:"chat_id,omitempty" db:"chat_id"`
	FreeSwitchName string           `json:"freeswitch_name" db:"freeswitch_name"`
	CreatedBy      uuid.UUID        `json:"created_by" db:"created_by"`
	Status         ConferenceStatus `json:"status" db:"status"`
	MaxMembers     int              `json:"max_members" db:"max_members"`
	IsPrivate      bool             `json:"is_private" db:"is_private"`
	RecordingPath  *string          `json:"recording_path,omitempty" db:"recording_path"`
	StartedAt      *time.Time       `json:"started_at,omitempty" db:"started_at"`
	EndedAt        *time.Time       `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at" db:"updated_at"`

	// Scheduled events fields
	EventType     EventType       `json:"event_type" db:"event_type"`
	ScheduledAt   *time.Time      `json:"scheduled_at,omitempty" db:"scheduled_at"`
	SeriesID      *uuid.UUID      `json:"series_id,omitempty" db:"series_id"`
	AcceptedCount int             `json:"accepted_count" db:"accepted_count"`
	DeclinedCount int             `json:"declined_count" db:"declined_count"`
	Recurrence    *RecurrenceRule `json:"recurrence,omitempty" db:"-"`

	// Computed/joined fields (not in DB)
	ParticipantCount int           `json:"participant_count" db:"participant_count"`
	Participants     []Participant `json:"participants,omitempty" db:"-"`
}

type ParticipantStatus string

const (
	ParticipantStatusConnecting ParticipantStatus = "connecting"
	ParticipantStatusJoined     ParticipantStatus = "joined"
	ParticipantStatusLeft       ParticipantStatus = "left"
	ParticipantStatusKicked     ParticipantStatus = "kicked"
)

type Participant struct {
	ID           uuid.UUID         `json:"id" db:"id"`
	ConferenceID uuid.UUID         `json:"conference_id" db:"conference_id"`
	UserID       uuid.UUID         `json:"user_id" db:"user_id"`
	FSMemberID   *string           `json:"fs_member_id,omitempty" db:"fs_member_id"`
	Status       ParticipantStatus `json:"status" db:"status"`
	IsMuted      bool              `json:"is_muted" db:"is_muted"`
	IsDeaf       bool              `json:"is_deaf" db:"is_deaf"`
	IsSpeaking   bool              `json:"is_speaking" db:"is_speaking"`
	JoinedAt     *time.Time        `json:"joined_at,omitempty" db:"joined_at"`
	LeftAt       *time.Time        `json:"left_at,omitempty" db:"left_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`

	// Scheduled events fields
	Role       ConferenceRole `json:"role" db:"role"`
	RSVPStatus RSVPStatus     `json:"rsvp_status" db:"rsvp_status"`
	RSVPAt     *time.Time     `json:"rsvp_at,omitempty" db:"rsvp_at"`

	// User info (joined from users table)
	Username    *string `json:"username,omitempty" db:"username"`
	DisplayName *string `json:"display_name,omitempty" db:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty" db:"avatar_url"`
}

type CreateConferenceRequest struct {
	Name            string
	ChatID          *uuid.UUID
	CreatedBy       uuid.UUID
	MaxMembers      int
	IsPrivate       bool
	EnableRecording bool
}

type JoinOptions struct {
	Muted bool
}

type VertoCredentials struct {
	UserID     uuid.UUID   `json:"user_id"`
	Login      string      `json:"login"`
	Password   string      `json:"password"`
	WSUrl      string      `json:"ws_url"`
	IceServers []IceServer `json:"ice_servers"`
	ExpiresAt  int64       `json:"expires_at"`
}

type IceServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// Scheduled events request types

type ScheduleConferenceRequest struct {
	Name             string
	ChatID           *uuid.UUID
	UserID           uuid.UUID
	ScheduledAt      time.Time
	Recurrence       *RecurrenceRule
	ParticipantIDs   []uuid.UUID
	MaxMembers       int
	EnableRecording  bool
}

type CreateAdHocFromChatRequest struct {
	ChatID         uuid.UUID
	UserID         uuid.UUID
	ParticipantIDs []uuid.UUID // Empty = all chat members
}

type UpdateRSVPRequest struct {
	ConferenceID uuid.UUID
	UserID       uuid.UUID
	RSVPStatus   RSVPStatus
}

type UpdateParticipantRoleRequest struct {
	ConferenceID uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
	NewRole      ConferenceRole
}

type AddParticipantsRequest struct {
	ConferenceID uuid.UUID
	ActorUserID  uuid.UUID
	UserIDs      []uuid.UUID
	DefaultRole  ConferenceRole
}

type RemoveParticipantRequest struct {
	ConferenceID uuid.UUID
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
}

// ConferenceReminder for scheduled notifications
type ConferenceReminder struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ConferenceID  uuid.UUID  `json:"conference_id" db:"conference_id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	RemindAt      time.Time  `json:"remind_at" db:"remind_at"`
	MinutesBefore int        `json:"minutes_before" db:"minutes_before"`
	Sent          bool       `json:"sent" db:"sent"`
	SentAt        *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`

	// Joined fields
	ConferenceName *string    `json:"conference_name,omitempty" db:"conference_name"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty" db:"scheduled_at"`
}

// MapChatRoleToConferenceRole converts chat role to conference role
func MapChatRoleToConferenceRole(chatRole string) ConferenceRole {
	switch chatRole {
	case "owner", "admin":
		return RoleOriginator
	case "moderator":
		return RoleModerator
	default:
		return RoleParticipant
	}
}

// CanChangeRole checks if actor can change target's role to newRole
func CanChangeRole(actorRole, targetRole, newRole ConferenceRole) bool {
	// Originator can change anyone's role
	if actorRole == RoleOriginator {
		return true
	}

	// Moderator can change participant, speaker, assistant roles
	// But cannot change originator or moderator, and cannot assign originator or moderator
	if actorRole == RoleModerator {
		if targetRole == RoleOriginator || targetRole == RoleModerator {
			return false
		}
		if newRole == RoleOriginator || newRole == RoleModerator {
			return false
		}
		return true
	}

	// Others cannot change roles
	return false
}
