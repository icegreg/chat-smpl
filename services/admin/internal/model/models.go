package model

import (
	"time"

	"github.com/google/uuid"
)

// ConferenceStatus represents conference status
type ConferenceStatus string

const (
	ConferenceStatusScheduled ConferenceStatus = "scheduled"
	ConferenceStatusActive    ConferenceStatus = "active"
	ConferenceStatusEnded     ConferenceStatus = "ended"
)

// EventType represents conference event type
type EventType string

const (
	EventTypeScheduled EventType = "scheduled"
	EventTypeAdhoc     EventType = "adhoc"
)

// ParticipantStatus represents participant status
type ParticipantStatus string

const (
	ParticipantStatusConnecting  ParticipantStatus = "connecting"
	ParticipantStatusConnected   ParticipantStatus = "connected"
	ParticipantStatusDisconnected ParticipantStatus = "disconnected"
)

// Conference represents a voice conference
type Conference struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	EventType   EventType         `json:"event_type"`
	ChatID      *uuid.UUID        `json:"chat_id,omitempty"`
	Status      ConferenceStatus  `json:"status"`
	CreatedBy   uuid.UUID         `json:"created_by"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	EndedAt     *time.Time        `json:"ended_at,omitempty"`
	Duration    *int64            `json:"duration_seconds,omitempty"`
	Participants int              `json:"participants_count"`
	Quality     string            `json:"quality,omitempty"`
}

// Participant represents a conference participant
type Participant struct {
	ID            uuid.UUID         `json:"id"`
	ConferenceID  uuid.UUID         `json:"conference_id"`
	UserID        uuid.UUID         `json:"user_id"`
	Username      string            `json:"username"`
	Extension     string            `json:"extension,omitempty"`
	Status        ParticipantStatus `json:"status"`
	JoinedAt      *time.Time        `json:"joined_at,omitempty"`
	LeftAt        *time.Time        `json:"left_at,omitempty"`
	Duration      *int64            `json:"duration_seconds,omitempty"`
	Device        string            `json:"device,omitempty"`
	Quality       string            `json:"quality,omitempty"`
}

// ServiceStatus represents status of a microservice
type ServiceStatus string

const (
	ServiceStatusRunning ServiceStatus = "running"
	ServiceStatusStopped ServiceStatus = "stopped"
	ServiceStatusError   ServiceStatus = "error"
	ServiceStatusUnknown ServiceStatus = "unknown"
)

// Service represents a microservice in the system
type Service struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Host        string        `json:"host,omitempty"`
	Port        int           `json:"port,omitempty"`
	Status      ServiceStatus `json:"status"`
	Health      string        `json:"health,omitempty"`
	Uptime      *int64        `json:"uptime_seconds,omitempty"`
	CPU         float64       `json:"cpu_percent,omitempty"`
	Memory      int64         `json:"memory_bytes,omitempty"`
	Connections int           `json:"connections,omitempty"`
	LastCheck   *time.Time    `json:"last_check,omitempty"`
}

// ConferencesResponse is the response for list conferences endpoint
type ConferencesResponse struct {
	Conferences []Conference `json:"conferences"`
	Total       int          `json:"total"`
}

// ParticipantsResponse is the response for list participants endpoint
type ParticipantsResponse struct {
	Participants []Participant `json:"participants"`
	Total        int           `json:"total"`
}

// ServicesResponse is the response for list services endpoint
type ServicesResponse struct {
	Services []Service `json:"services"`
	Total    int       `json:"total"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
