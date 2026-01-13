package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/icegreg/chat-smpl/services/voice/internal/model"
)

const (
	exchangeName = "voice.events"
	exchangeType = "topic"
)

// Event routing keys
const (
	ConferenceCreatedKey    = "conference.created"
	ConferenceEndedKey      = "conference.ended"
	ParticipantJoinedKey    = "participant.joined"
	ParticipantLeftKey      = "participant.left"
	ParticipantMutedKey     = "participant.muted"
	ParticipantSpeakingKey  = "participant.speaking"
	CallInitiatedKey        = "call.initiated"
	CallAnsweredKey         = "call.answered"
	CallEndedKey            = "call.ended"

	// Scheduled events
	ConferenceScheduledKey      = "conference.scheduled"
	ConferenceCancelledKey      = "conference.cancelled"
	ConferenceRSVPUpdatedKey    = "conference.rsvp_updated"
	ParticipantRoleChangedKey   = "participant.role_changed"
	ParticipantAddedKey         = "participant.added"
	ParticipantRemovedKey       = "participant.removed"
	ConferenceReminderKey       = "conference.reminder"
)

// Publisher interface for voice events
type Publisher interface {
	PublishConferenceCreated(ctx context.Context, conf *model.Conference) error
	PublishConferenceEnded(ctx context.Context, conf *model.Conference) error
	PublishParticipantJoined(ctx context.Context, p *model.Participant, chatID string) error
	PublishParticipantLeft(ctx context.Context, p *model.Participant, chatID string) error
	PublishParticipantMuted(ctx context.Context, p *model.Participant, chatID string) error
	PublishParticipantSpeaking(ctx context.Context, participantID string, isSpeaking bool) error
	PublishCallInitiated(ctx context.Context, call *model.Call) error
	PublishCallAnswered(ctx context.Context, call *model.Call) error
	PublishCallEnded(ctx context.Context, call *model.Call) error

	// Scheduled events
	PublishConferenceScheduled(ctx context.Context, conf *model.Conference) error
	PublishConferenceCancelled(ctx context.Context, conf *model.Conference) error
	PublishRSVPUpdated(ctx context.Context, confID, userID string, status model.RSVPStatus) error
	PublishParticipantRoleChanged(ctx context.Context, confID, userID string, oldRole, newRole model.ConferenceRole) error
	PublishParticipantAdded(ctx context.Context, p *model.Participant, chatID string) error
	PublishParticipantRemoved(ctx context.Context, confID, userID string) error
	PublishConferenceReminder(ctx context.Context, reminder *model.ConferenceReminder) error

	Close() error
}

type publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *zap.Logger
}

// NewPublisher creates a new RabbitMQ publisher for voice events
func NewPublisher(amqpURL string, logger *zap.Logger) (Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	if err := ch.ExchangeDeclare(
		exchangeName,
		exchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info("voice events publisher initialized", zap.String("exchange", exchangeName))

	return &publisher{
		conn:    conn,
		channel: ch,
		logger:  logger,
	}, nil
}

// Close closes the RabbitMQ connection
func (p *publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

func (p *publisher) publish(ctx context.Context, routingKey string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("published voice event",
		zap.String("routingKey", routingKey),
		zap.String("body", string(body)))

	return nil
}

// ConferenceEvent represents a conference event payload
type ConferenceEvent struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ChatID           *string   `json:"chat_id,omitempty"`
	CreatedBy        string    `json:"created_by"`
	Status           string    `json:"status"`
	MaxMembers       int       `json:"max_members"`
	ParticipantCount int       `json:"participant_count"`
	StartedAt        *string   `json:"started_at,omitempty"`
	EndedAt          *string   `json:"ended_at,omitempty"`
	CreatedAt        string    `json:"created_at"`
}

func conferenceToEvent(conf *model.Conference) ConferenceEvent {
	event := ConferenceEvent{
		ID:               conf.ID.String(),
		Name:             conf.Name,
		CreatedBy:        conf.CreatedBy.String(),
		Status:           string(conf.Status),
		MaxMembers:       conf.MaxMembers,
		ParticipantCount: conf.ParticipantCount,
		CreatedAt:        conf.CreatedAt.Format(time.RFC3339),
	}
	if conf.ChatID != nil {
		chatID := conf.ChatID.String()
		event.ChatID = &chatID
	}
	if conf.StartedAt != nil {
		startedAt := conf.StartedAt.Format(time.RFC3339)
		event.StartedAt = &startedAt
	}
	if conf.EndedAt != nil {
		endedAt := conf.EndedAt.Format(time.RFC3339)
		event.EndedAt = &endedAt
	}
	return event
}

// PublishConferenceCreated publishes conference.created event
func (p *publisher) PublishConferenceCreated(ctx context.Context, conf *model.Conference) error {
	return p.publish(ctx, ConferenceCreatedKey, conferenceToEvent(conf))
}

// PublishConferenceEnded publishes conference.ended event
func (p *publisher) PublishConferenceEnded(ctx context.Context, conf *model.Conference) error {
	return p.publish(ctx, ConferenceEndedKey, conferenceToEvent(conf))
}

// ParticipantEvent represents a participant event payload
type ParticipantEvent struct {
	ID           string  `json:"id"`
	ConferenceID string  `json:"conference_id"`
	ChatID       string  `json:"chat_id,omitempty"`
	UserID       string  `json:"user_id"`
	Status       string  `json:"status"`
	IsMuted      bool    `json:"is_muted"`
	IsDeaf       bool    `json:"is_deaf"`
	IsSpeaking   bool    `json:"is_speaking"`
	Username     *string `json:"username,omitempty"`
	DisplayName  *string `json:"display_name,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	JoinedAt     *string `json:"joined_at,omitempty"`
	LeftAt       *string `json:"left_at,omitempty"`
}

func participantToEvent(p *model.Participant, chatID string) ParticipantEvent {
	event := ParticipantEvent{
		ID:           p.ID.String(),
		ConferenceID: p.ConferenceID.String(),
		ChatID:       chatID,
		UserID:       p.UserID.String(),
		Status:       string(p.Status),
		IsMuted:      p.IsMuted,
		IsDeaf:       p.IsDeaf,
		IsSpeaking:   p.IsSpeaking,
		Username:     p.Username,
		DisplayName:  p.DisplayName,
		AvatarURL:    p.AvatarURL,
	}
	if p.JoinedAt != nil {
		joinedAt := p.JoinedAt.Format(time.RFC3339)
		event.JoinedAt = &joinedAt
	}
	if p.LeftAt != nil {
		leftAt := p.LeftAt.Format(time.RFC3339)
		event.LeftAt = &leftAt
	}
	return event
}

// PublishParticipantJoined publishes participant.joined event
func (p *publisher) PublishParticipantJoined(ctx context.Context, participant *model.Participant, chatID string) error {
	return p.publish(ctx, ParticipantJoinedKey, participantToEvent(participant, chatID))
}

// PublishParticipantLeft publishes participant.left event
func (p *publisher) PublishParticipantLeft(ctx context.Context, participant *model.Participant, chatID string) error {
	return p.publish(ctx, ParticipantLeftKey, participantToEvent(participant, chatID))
}

// PublishParticipantMuted publishes participant.muted event
func (p *publisher) PublishParticipantMuted(ctx context.Context, participant *model.Participant, chatID string) error {
	return p.publish(ctx, ParticipantMutedKey, participantToEvent(participant, chatID))
}

// SpeakingEvent represents a speaking state change
type SpeakingEvent struct {
	ParticipantID string `json:"participant_id"`
	IsSpeaking    bool   `json:"is_speaking"`
}

// PublishParticipantSpeaking publishes participant.speaking event
func (p *publisher) PublishParticipantSpeaking(ctx context.Context, participantID string, isSpeaking bool) error {
	return p.publish(ctx, ParticipantSpeakingKey, SpeakingEvent{
		ParticipantID: participantID,
		IsSpeaking:    isSpeaking,
	})
}

// CallEvent represents a call event payload
type CallEvent struct {
	ID                string  `json:"id"`
	CallerID          string  `json:"caller_id"`
	CalleeID          string  `json:"callee_id"`
	ChatID            *string `json:"chat_id,omitempty"`
	ConferenceID      *string `json:"conference_id,omitempty"`
	Status            string  `json:"status"`
	Duration          int     `json:"duration"`
	EndReason         *string `json:"end_reason,omitempty"`
	CallerUsername    *string `json:"caller_username,omitempty"`
	CallerDisplayName *string `json:"caller_display_name,omitempty"`
	CalleeUsername    *string `json:"callee_username,omitempty"`
	CalleeDisplayName *string `json:"callee_display_name,omitempty"`
	StartedAt         *string `json:"started_at,omitempty"`
	AnsweredAt        *string `json:"answered_at,omitempty"`
	EndedAt           *string `json:"ended_at,omitempty"`
}

func callToEvent(call *model.Call) CallEvent {
	event := CallEvent{
		ID:                call.ID.String(),
		CallerID:          call.CallerID.String(),
		CalleeID:          call.CalleeID.String(),
		Status:            string(call.Status),
		Duration:          call.Duration,
		EndReason:         call.EndReason,
		CallerUsername:    call.CallerUsername,
		CallerDisplayName: call.CallerDisplayName,
		CalleeUsername:    call.CalleeUsername,
		CalleeDisplayName: call.CalleeDisplayName,
	}
	if call.ChatID != nil {
		chatID := call.ChatID.String()
		event.ChatID = &chatID
	}
	if call.ConferenceID != nil {
		confID := call.ConferenceID.String()
		event.ConferenceID = &confID
	}
	if call.StartedAt != nil {
		startedAt := call.StartedAt.Format(time.RFC3339)
		event.StartedAt = &startedAt
	}
	if call.AnsweredAt != nil {
		answeredAt := call.AnsweredAt.Format(time.RFC3339)
		event.AnsweredAt = &answeredAt
	}
	if call.EndedAt != nil {
		endedAt := call.EndedAt.Format(time.RFC3339)
		event.EndedAt = &endedAt
	}
	return event
}

// PublishCallInitiated publishes call.initiated event
func (p *publisher) PublishCallInitiated(ctx context.Context, call *model.Call) error {
	return p.publish(ctx, CallInitiatedKey, callToEvent(call))
}

// PublishCallAnswered publishes call.answered event
func (p *publisher) PublishCallAnswered(ctx context.Context, call *model.Call) error {
	return p.publish(ctx, CallAnsweredKey, callToEvent(call))
}

// PublishCallEnded publishes call.ended event
func (p *publisher) PublishCallEnded(ctx context.Context, call *model.Call) error {
	return p.publish(ctx, CallEndedKey, callToEvent(call))
}

// ScheduledConferenceEvent represents a scheduled conference event payload
type ScheduledConferenceEvent struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	ChatID           *string `json:"chat_id,omitempty"`
	CreatedBy        string  `json:"created_by"`
	Status           string  `json:"status"`
	EventType        string  `json:"event_type"`
	ScheduledAt      *string `json:"scheduled_at,omitempty"`
	SeriesID         *string `json:"series_id,omitempty"`
	AcceptedCount    int     `json:"accepted_count"`
	DeclinedCount    int     `json:"declined_count"`
	ParticipantCount int     `json:"participant_count"`
	CreatedAt        string  `json:"created_at"`
}

func scheduledConferenceToEvent(conf *model.Conference) ScheduledConferenceEvent {
	event := ScheduledConferenceEvent{
		ID:               conf.ID.String(),
		Name:             conf.Name,
		CreatedBy:        conf.CreatedBy.String(),
		Status:           string(conf.Status),
		EventType:        string(conf.EventType),
		AcceptedCount:    conf.AcceptedCount,
		DeclinedCount:    conf.DeclinedCount,
		ParticipantCount: conf.ParticipantCount,
		CreatedAt:        conf.CreatedAt.Format(time.RFC3339),
	}
	if conf.ChatID != nil {
		chatID := conf.ChatID.String()
		event.ChatID = &chatID
	}
	if conf.ScheduledAt != nil {
		scheduledAt := conf.ScheduledAt.Format(time.RFC3339)
		event.ScheduledAt = &scheduledAt
	}
	if conf.SeriesID != nil {
		seriesID := conf.SeriesID.String()
		event.SeriesID = &seriesID
	}
	return event
}

// PublishConferenceScheduled publishes conference.scheduled event
func (p *publisher) PublishConferenceScheduled(ctx context.Context, conf *model.Conference) error {
	return p.publish(ctx, ConferenceScheduledKey, scheduledConferenceToEvent(conf))
}

// PublishConferenceCancelled publishes conference.cancelled event
func (p *publisher) PublishConferenceCancelled(ctx context.Context, conf *model.Conference) error {
	return p.publish(ctx, ConferenceCancelledKey, scheduledConferenceToEvent(conf))
}

// RSVPUpdatedEvent represents RSVP update event
type RSVPUpdatedEvent struct {
	ConferenceID string `json:"conference_id"`
	UserID       string `json:"user_id"`
	RSVPStatus   string `json:"rsvp_status"`
}

// PublishRSVPUpdated publishes conference.rsvp_updated event
func (p *publisher) PublishRSVPUpdated(ctx context.Context, confID, userID string, status model.RSVPStatus) error {
	return p.publish(ctx, ConferenceRSVPUpdatedKey, RSVPUpdatedEvent{
		ConferenceID: confID,
		UserID:       userID,
		RSVPStatus:   string(status),
	})
}

// ParticipantRoleChangedEvent represents role change event
type ParticipantRoleChangedEvent struct {
	ConferenceID string `json:"conference_id"`
	UserID       string `json:"user_id"`
	OldRole      string `json:"old_role"`
	NewRole      string `json:"new_role"`
}

// PublishParticipantRoleChanged publishes participant.role_changed event
func (p *publisher) PublishParticipantRoleChanged(ctx context.Context, confID, userID string, oldRole, newRole model.ConferenceRole) error {
	return p.publish(ctx, ParticipantRoleChangedKey, ParticipantRoleChangedEvent{
		ConferenceID: confID,
		UserID:       userID,
		OldRole:      string(oldRole),
		NewRole:      string(newRole),
	})
}

// PublishParticipantAdded publishes participant.added event
func (p *publisher) PublishParticipantAdded(ctx context.Context, participant *model.Participant, chatID string) error {
	return p.publish(ctx, ParticipantAddedKey, participantToEvent(participant, chatID))
}

// ParticipantRemovedEvent represents participant removal event
type ParticipantRemovedEvent struct {
	ConferenceID string `json:"conference_id"`
	UserID       string `json:"user_id"`
}

// PublishParticipantRemoved publishes participant.removed event
func (p *publisher) PublishParticipantRemoved(ctx context.Context, confID, userID string) error {
	return p.publish(ctx, ParticipantRemovedKey, ParticipantRemovedEvent{
		ConferenceID: confID,
		UserID:       userID,
	})
}

// ConferenceReminderEventPayload represents reminder event payload
type ConferenceReminderEventPayload struct {
	ConferenceID   string `json:"conference_id"`
	UserID         string `json:"user_id"`
	ConferenceName string `json:"conference_name"`
	ScheduledAt    string `json:"scheduled_at"`
	MinutesBefore  int    `json:"minutes_before"`
}

// PublishConferenceReminder publishes conference.reminder event
func (p *publisher) PublishConferenceReminder(ctx context.Context, reminder *model.ConferenceReminder) error {
	scheduledAt := ""
	if reminder.ScheduledAt != nil {
		scheduledAt = reminder.ScheduledAt.Format(time.RFC3339)
	}
	conferenceName := ""
	if reminder.ConferenceName != nil {
		conferenceName = *reminder.ConferenceName
	}

	return p.publish(ctx, ConferenceReminderKey, ConferenceReminderEventPayload{
		ConferenceID:   reminder.ConferenceID.String(),
		UserID:         reminder.UserID.String(),
		ConferenceName: conferenceName,
		ScheduledAt:    scheduledAt,
		MinutesBefore:  reminder.MinutesBefore,
	})
}
