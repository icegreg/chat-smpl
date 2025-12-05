package model

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

func (t ChatType) IsValid() bool {
	switch t {
	case ChatTypePrivate, ChatTypeGroup, ChatTypeChannel:
		return true
	}
	return false
}

type ParticipantRole string

const (
	ParticipantRoleAdmin    ParticipantRole = "admin"
	ParticipantRoleMember   ParticipantRole = "member"
	ParticipantRoleReadonly ParticipantRole = "readonly"
)

func (r ParticipantRole) IsValid() bool {
	switch r {
	case ParticipantRoleAdmin, ParticipantRoleMember, ParticipantRoleReadonly:
		return true
	}
	return false
}

func (r ParticipantRole) CanWrite() bool {
	return r == ParticipantRoleAdmin || r == ParticipantRoleMember
}

func (r ParticipantRole) CanModerate() bool {
	return r == ParticipantRoleAdmin
}

type Chat struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ChatType  ChatType  `json:"chat_type" db:"chat_type"`
	CreatedBy uuid.UUID `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ChatParticipant struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ChatID      uuid.UUID       `json:"chat_id" db:"chat_id"`
	UserID      uuid.UUID       `json:"user_id" db:"user_id"`
	Role        ParticipantRole `json:"role" db:"role"`
	JoinedAt    time.Time       `json:"joined_at" db:"joined_at"`
	Username    *string         `json:"username,omitempty" db:"username"`
	Email       *string         `json:"email,omitempty" db:"email"`
	DisplayName *string         `json:"display_name,omitempty" db:"display_name"`
	AvatarURL   *string         `json:"avatar_url,omitempty" db:"avatar_url"`
}

type Message struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	ChatID            uuid.UUID  `json:"chat_id" db:"chat_id"`
	ParentID          *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	SenderID          uuid.UUID  `json:"sender_id" db:"sender_id"`
	Content           string     `json:"content" db:"content"`
	SentAt            time.Time  `json:"sent_at" db:"sent_at"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	IsDeleted         bool       `json:"is_deleted" db:"is_deleted"`
	SenderUsername    *string    `json:"sender_username,omitempty" db:"sender_username"`
	SenderDisplayName *string    `json:"sender_display_name,omitempty" db:"sender_display_name"`
	SenderAvatarURL   *string    `json:"sender_avatar_url,omitempty" db:"sender_avatar_url"`
}

type Reaction struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Reaction  string    `json:"reaction" db:"reaction"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type MessageReader struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MessageID uuid.UUID `json:"message_id" db:"message_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	ReadAt    time.Time `json:"read_at" db:"read_at"`
}

type Poll struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	ChatID           uuid.UUID  `json:"chat_id" db:"chat_id"`
	MessageID        *uuid.UUID `json:"message_id,omitempty" db:"message_id"`
	CreatedBy        uuid.UUID  `json:"created_by" db:"created_by"`
	Question         string     `json:"question" db:"question"`
	IsMultipleChoice bool       `json:"is_multiple_choice" db:"is_multiple_choice"`
	IsAnonymous      bool       `json:"is_anonymous" db:"is_anonymous"`
	IsFinished       bool       `json:"is_finished" db:"is_finished"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty" db:"finished_at"`
}

type PollOption struct {
	ID        uuid.UUID `json:"id" db:"id"`
	PollID    uuid.UUID `json:"poll_id" db:"poll_id"`
	Text      string    `json:"text" db:"text"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
}

type PollVote struct {
	ID       uuid.UUID `json:"id" db:"id"`
	PollID   uuid.UUID `json:"poll_id" db:"poll_id"`
	OptionID uuid.UUID `json:"option_id" db:"option_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	VotedAt  time.Time `json:"voted_at" db:"voted_at"`
}

type ChatFavorite struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ChatID    uuid.UUID `json:"chat_id" db:"chat_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ArchivedChat struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ChatID     uuid.UUID `json:"chat_id" db:"chat_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	ArchivedAt time.Time `json:"archived_at" db:"archived_at"`
}
