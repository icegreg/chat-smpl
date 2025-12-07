package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/icegreg/chat-smpl/services/presence/internal/repository"
)

const (
	exchangeName = "presence.events"
)

// PresenceEvent represents a presence change event
type PresenceEvent struct {
	Type      string    `json:"type"` // presence.changed
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
	Data      any       `json:"data"`
}

// PresenceData is the data in presence events
type PresenceData struct {
	Status          string `json:"status"`
	IsOnline        bool   `json:"is_online"`
	ConnectionCount int    `json:"connection_count"`
	LastSeenAt      string `json:"last_seen_at,omitempty"`
}

// Service handles presence business logic
type Service struct {
	repo      *repository.Repository
	publisher *amqp.Channel
}

// NewService creates a new presence service
func NewService(repo *repository.Repository, publisher *amqp.Channel) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}

// SetupExchange declares the presence exchange
func (s *Service) SetupExchange() error {
	if s.publisher == nil {
		return nil // No publisher configured
	}
	return s.publisher.ExchangeDeclare(
		exchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

// SetStatus sets user's status and publishes event
func (s *Service) SetStatus(ctx context.Context, userID string, status repository.UserStatus) (*repository.PresenceInfo, error) {
	if err := s.repo.SetStatus(ctx, userID, status); err != nil {
		return nil, fmt.Errorf("failed to set status: %w", err)
	}

	presence, err := s.repo.GetPresence(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}

	// Publish presence change event
	if err := s.publishPresenceChange(presence); err != nil {
		// Log but don't fail - status was already saved
		fmt.Printf("failed to publish presence change: %v\n", err)
	}

	return presence, nil
}

// GetPresence returns presence info for a user
func (s *Service) GetPresence(ctx context.Context, userID string) (*repository.PresenceInfo, error) {
	return s.repo.GetPresence(ctx, userID)
}

// GetPresencesBatch returns presence info for multiple users
func (s *Service) GetPresencesBatch(ctx context.Context, userIDs []string) ([]*repository.PresenceInfo, error) {
	return s.repo.GetPresencesBatch(ctx, userIDs)
}

// UserConnected is called when a user establishes websocket connection
func (s *Service) UserConnected(ctx context.Context, userID, connectionID string) (*repository.PresenceInfo, error) {
	// Check if this is the first connection (was offline)
	wasOnline, err := s.repo.IsOnline(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check online status: %w", err)
	}

	// Add connection
	if err := s.repo.AddConnection(ctx, userID, connectionID); err != nil {
		return nil, fmt.Errorf("failed to add connection: %w", err)
	}

	presence, err := s.repo.GetPresence(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}

	// Publish event only if user just came online
	if !wasOnline && presence.IsOnline {
		if err := s.publishPresenceChange(presence); err != nil {
			fmt.Printf("failed to publish presence change: %v\n", err)
		}
	}

	return presence, nil
}

// UserDisconnected is called when a user's websocket connection closes
func (s *Service) UserDisconnected(ctx context.Context, userID, connectionID string) (*repository.PresenceInfo, error) {
	// Remove connection
	if err := s.repo.RemoveConnection(ctx, userID, connectionID); err != nil {
		return nil, fmt.Errorf("failed to remove connection: %w", err)
	}

	presence, err := s.repo.GetPresence(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}

	// If user went offline, publish event
	if !presence.IsOnline {
		if err := s.publishPresenceChange(presence); err != nil {
			fmt.Printf("failed to publish presence change: %v\n", err)
		}
	}

	return presence, nil
}

func (s *Service) publishPresenceChange(presence *repository.PresenceInfo) error {
	if s.publisher == nil {
		return nil // No publisher configured
	}

	lastSeenStr := ""
	if !presence.LastSeenAt.IsZero() {
		lastSeenStr = presence.LastSeenAt.Format(time.RFC3339)
	}

	event := PresenceEvent{
		Type:      "presence.changed",
		Timestamp: time.Now(),
		UserID:    presence.UserID,
		Data: PresenceData{
			Status:          string(presence.Status),
			IsOnline:        presence.IsOnline,
			ConnectionCount: presence.ConnectionCount,
			LastSeenAt:      lastSeenStr,
		},
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return s.publisher.PublishWithContext(
		context.Background(),
		exchangeName,
		"presence.changed",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}
