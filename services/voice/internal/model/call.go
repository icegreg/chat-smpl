package model

import (
	"time"

	"github.com/google/uuid"
)

type CallStatus string

const (
	CallStatusInitiated CallStatus = "initiated"
	CallStatusRinging   CallStatus = "ringing"
	CallStatusAnswered  CallStatus = "answered"
	CallStatusEnded     CallStatus = "ended"
	CallStatusFailed    CallStatus = "failed"
	CallStatusMissed    CallStatus = "missed"
)

type Call struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CallerID     uuid.UUID  `json:"caller_id" db:"caller_id"`
	CalleeID     uuid.UUID  `json:"callee_id" db:"callee_id"`
	ChatID       *uuid.UUID `json:"chat_id,omitempty" db:"chat_id"`
	ConferenceID *uuid.UUID `json:"conference_id,omitempty" db:"conference_id"`
	Status       CallStatus `json:"status" db:"status"`
	FSCallUUID   *string    `json:"fs_call_uuid,omitempty" db:"fs_call_uuid"`
	Duration     int        `json:"duration" db:"duration"`
	EndReason    *string    `json:"end_reason,omitempty" db:"end_reason"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	AnsweredAt   *time.Time `json:"answered_at,omitempty" db:"answered_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`

	// User info (joined from users table)
	CallerUsername    *string `json:"caller_username,omitempty" db:"caller_username"`
	CallerDisplayName *string `json:"caller_display_name,omitempty" db:"caller_display_name"`
	CalleeUsername    *string `json:"callee_username,omitempty" db:"callee_username"`
	CalleeDisplayName *string `json:"callee_display_name,omitempty" db:"callee_display_name"`
}

type CallHistory struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	CallerID     uuid.UUID  `json:"caller_id" db:"caller_id"`
	CalleeID     uuid.UUID  `json:"callee_id" db:"callee_id"`
	ChatID       *uuid.UUID `json:"chat_id,omitempty" db:"chat_id"`
	Status       CallStatus `json:"status" db:"status"`
	Duration     int        `json:"duration" db:"duration"`
	EndReason    *string    `json:"end_reason,omitempty" db:"end_reason"`
	StartedAt    *time.Time `json:"started_at,omitempty" db:"started_at"`
	AnsweredAt   *time.Time `json:"answered_at,omitempty" db:"answered_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty" db:"ended_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`

	// User info
	CallerUsername    *string `json:"caller_username,omitempty" db:"caller_username"`
	CallerDisplayName *string `json:"caller_display_name,omitempty" db:"caller_display_name"`
	CalleeUsername    *string `json:"callee_username,omitempty" db:"callee_username"`
	CalleeDisplayName *string `json:"callee_display_name,omitempty" db:"callee_display_name"`

	// Direction relative to user
	Direction string `json:"direction" db:"direction"` // "inbound" or "outbound"
}
