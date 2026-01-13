package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	pb "github.com/icegreg/chat-smpl/proto/chat"
	"github.com/icegreg/chat-smpl/services/websocket/internal/centrifugo"
	"go.uber.org/zap"
)

const (
	VoiceExchangeName = "voice.events"
	VoiceQueueName    = "websocket.voice.events"
	VoiceConsumerName = "websocket-voice-service"
)

// VoiceEvent represents events from voice-service
type VoiceEvent struct {
	// Common fields
	ID        string `json:"id"`
	Timestamp string `json:"timestamp,omitempty"`

	// Conference fields
	ConferenceID     string `json:"conference_id,omitempty"`
	Name             string `json:"name,omitempty"`
	ChatID           string `json:"chat_id,omitempty"`
	CreatedBy        string `json:"created_by,omitempty"`
	Status           string `json:"status,omitempty"`
	MaxMembers       int    `json:"max_members,omitempty"`
	ParticipantCount int    `json:"participant_count,omitempty"`

	// Participant fields
	UserID      string `json:"user_id,omitempty"`
	IsMuted     bool   `json:"is_muted,omitempty"`
	IsDeaf      bool   `json:"is_deaf,omitempty"`
	IsSpeaking  bool   `json:"is_speaking,omitempty"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`

	// Call fields
	CallerID          string `json:"caller_id,omitempty"`
	CalleeID          string `json:"callee_id,omitempty"`
	Duration          int    `json:"duration,omitempty"`
	EndReason         string `json:"end_reason,omitempty"`
	CallerUsername    string `json:"caller_username,omitempty"`
	CallerDisplayName string `json:"caller_display_name,omitempty"`
	CalleeUsername    string `json:"callee_username,omitempty"`
	CalleeDisplayName string `json:"callee_display_name,omitempty"`
}

type VoiceConsumer struct {
	rmqConn    *rabbitmq.Connection
	centrifugo *centrifugo.Client
	chatClient pb.ChatServiceClient
}

func NewVoiceConsumer(rmqConn *rabbitmq.Connection, centrifugoClient *centrifugo.Client, chatClient pb.ChatServiceClient) *VoiceConsumer {
	return &VoiceConsumer{
		rmqConn:    rmqConn,
		centrifugo: centrifugoClient,
		chatClient: chatClient,
	}
}

func (c *VoiceConsumer) Setup() error {
	// Declare exchange if not exists
	if err := c.rmqConn.DeclareExchange(rabbitmq.Exchange{
		Name:       VoiceExchangeName,
		Kind:       "topic",
		Durable:    true,
		AutoDelete: false,
	}); err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err := c.rmqConn.DeclareQueue(rabbitmq.Queue{
		Name:       VoiceQueueName,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
	})
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange with patterns
	patterns := []string{
		"conference.#",
		"participant.#",
		"call.#",
	}

	for _, pattern := range patterns {
		if err := c.rmqConn.BindQueue(VoiceQueueName, pattern, VoiceExchangeName); err != nil {
			return fmt.Errorf("failed to bind queue with pattern %s: %w", pattern, err)
		}
	}

	logger.Info("voice consumer setup complete",
		zap.String("queue", VoiceQueueName),
		zap.String("exchange", VoiceExchangeName),
	)

	return nil
}

func (c *VoiceConsumer) Start(ctx context.Context) error {
	consumer := rabbitmq.NewConsumer(
		c.rmqConn,
		VoiceQueueName,
		VoiceConsumerName,
		rabbitmq.WithPrefetch(50),
		rabbitmq.WithWorkers(5),
	)

	logger.Info("starting voice consumer",
		zap.String("queue", VoiceQueueName),
		zap.Int("prefetch", 50),
		zap.Int("workers", 5),
	)

	return consumer.Consume(ctx, c.handleMessage)
}

func (c *VoiceConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	routingKey := msg.RoutingKey

	logger.Debug("received voice event",
		zap.String("routing_key", routingKey),
	)

	var event VoiceEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		logger.Error("failed to unmarshal voice event",
			zap.Error(err),
			zap.String("routing_key", routingKey),
		)
		return nil // Don't retry malformed messages
	}

	// Determine event type and recipients based on routing key
	switch {
	case routingKey == "conference.created" || routingKey == "conference.ended":
		return c.handleConferenceEvent(ctx, routingKey, event, msg.Body)

	case routingKey == "participant.joined" || routingKey == "participant.left" ||
		routingKey == "participant.muted" || routingKey == "participant.speaking":
		return c.handleParticipantEvent(ctx, routingKey, event, msg.Body)

	case routingKey == "call.initiated" || routingKey == "call.answered" || routingKey == "call.ended":
		return c.handleCallEvent(ctx, routingKey, event, msg.Body)

	default:
		logger.Warn("unknown voice event type", zap.String("routing_key", routingKey))
		return nil
	}
}

func (c *VoiceConsumer) handleConferenceEvent(ctx context.Context, eventType string, event VoiceEvent, rawData []byte) error {
	// Conference events are sent to the conference channel if chat_id exists,
	// otherwise to the creator's personal channel
	voiceEvent := map[string]interface{}{
		"type": eventType,
		"data": json.RawMessage(rawData),
	}

	if event.ChatID != "" {
		// Broadcast to chat channel - all chat participants will receive it
		channel := fmt.Sprintf("chat:%s", event.ChatID)
		if err := c.centrifugo.PublishToChannel(ctx, channel, voiceEvent); err != nil {
			logger.Error("failed to publish conference event to chat channel",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.String("chat_id", event.ChatID),
			)
			return err
		}
		logger.Debug("conference event published to chat",
			zap.String("event_type", eventType),
			zap.String("chat_id", event.ChatID),
		)
	}

	// Also send to conference-specific channel for direct subscribers
	if event.ID != "" {
		channel := fmt.Sprintf("conference:%s", event.ID)
		if err := c.centrifugo.PublishToChannel(ctx, channel, voiceEvent); err != nil {
			logger.Error("failed to publish conference event to conference channel",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.String("conference_id", event.ID),
			)
			return err
		}
	}

	return nil
}

func (c *VoiceConsumer) handleParticipantEvent(ctx context.Context, eventType string, event VoiceEvent, rawData []byte) error {
	voiceEvent := map[string]interface{}{
		"type": eventType,
		"data": json.RawMessage(rawData),
	}

	// Send to conference channel
	if event.ConferenceID != "" {
		channel := fmt.Sprintf("conference:%s", event.ConferenceID)
		if err := c.centrifugo.PublishToChannel(ctx, channel, voiceEvent); err != nil {
			logger.Error("failed to publish participant event",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.String("conference_id", event.ConferenceID),
			)
			return err
		}
	}

	// Also send to user's personal channel
	if event.UserID != "" {
		if err := c.centrifugo.PublishToUser(ctx, event.UserID, voiceEvent); err != nil {
			logger.Error("failed to publish participant event to user",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.String("user_id", event.UserID),
			)
			return err
		}
	}

	// Send system message to chat when user joins/leaves conference
	if c.chatClient != nil && event.ChatID != "" {
		userName := event.DisplayName
		if userName == "" {
			userName = event.Username
		}
		if userName == "" {
			userName = "User"
		}

		var content string
		switch eventType {
		case "participant.joined":
			content = fmt.Sprintf("%s joined the conference", userName)
		case "participant.left":
			content = fmt.Sprintf("%s left the conference", userName)
		}

		if content != "" {
			_, err := c.chatClient.SendSystemMessage(ctx, &pb.SendSystemMessageRequest{
				ChatId:  event.ChatID,
				Content: content,
			})
			if err != nil {
				logger.Warn("failed to send system message to chat",
					zap.Error(err),
					zap.String("chat_id", event.ChatID),
					zap.String("content", content),
				)
			} else {
				logger.Debug("system message sent to chat",
					zap.String("chat_id", event.ChatID),
					zap.String("content", content),
				)
			}
		}
	}

	logger.Debug("participant event published",
		zap.String("event_type", eventType),
		zap.String("conference_id", event.ConferenceID),
		zap.String("user_id", event.UserID),
	)

	return nil
}

func (c *VoiceConsumer) handleCallEvent(ctx context.Context, eventType string, event VoiceEvent, rawData []byte) error {
	voiceEvent := map[string]interface{}{
		"type": eventType,
		"data": json.RawMessage(rawData),
	}

	// For call events, send to both caller and callee
	var recipients []string
	if event.CallerID != "" {
		recipients = append(recipients, event.CallerID)
	}
	if event.CalleeID != "" {
		recipients = append(recipients, event.CalleeID)
	}

	if len(recipients) > 0 {
		if err := c.centrifugo.BroadcastToUsers(ctx, recipients, voiceEvent); err != nil {
			logger.Error("failed to broadcast call event",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.Strings("recipients", recipients),
			)
			return err
		}
	}

	logger.Debug("call event broadcasted",
		zap.String("event_type", eventType),
		zap.String("call_id", event.ID),
		zap.Strings("recipients", recipients),
	)

	return nil
}
