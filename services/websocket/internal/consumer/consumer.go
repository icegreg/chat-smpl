package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	"github.com/icegreg/chat-smpl/services/websocket/internal/centrifugo"
	"go.uber.org/zap"
)

const (
	ExchangeName = "chat.events"
	QueueName    = "websocket.events"
	ConsumerName = "websocket-service"
)

// ChatEvent matches the structure from chat-service publisher
type ChatEvent struct {
	Type         string          `json:"type"`
	Timestamp    string          `json:"timestamp"`
	ActorID      string          `json:"actor_id"`
	ChatID       string          `json:"chat_id"`
	Participants []string        `json:"participants"`
	Data         json.RawMessage `json:"data"`
}

type Consumer struct {
	rmqConn    *rabbitmq.Connection
	centrifugo *centrifugo.Client
}

func New(rmqConn *rabbitmq.Connection, centrifugoClient *centrifugo.Client) *Consumer {
	return &Consumer{
		rmqConn:    rmqConn,
		centrifugo: centrifugoClient,
	}
}

func (c *Consumer) Setup() error {
	// Declare queue
	_, err := c.rmqConn.DeclareQueue(rabbitmq.Queue{
		Name:       QueueName,
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
		"chat.#",
		"message.#",
		"typing",
		"reaction.#",
	}

	for _, pattern := range patterns {
		if err := c.rmqConn.BindQueue(QueueName, pattern, ExchangeName); err != nil {
			return fmt.Errorf("failed to bind queue with pattern %s: %w", pattern, err)
		}
	}

	logger.Info("consumer setup complete",
		zap.String("queue", QueueName),
		zap.String("exchange", ExchangeName),
	)

	return nil
}

func (c *Consumer) Start(ctx context.Context) error {
	consumer := rabbitmq.NewConsumer(c.rmqConn, QueueName, ConsumerName)

	logger.Info("starting to consume messages", zap.String("queue", QueueName))

	return consumer.Consume(ctx, c.handleMessage)
}

func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var event ChatEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		logger.Error("failed to unmarshal event",
			zap.Error(err),
			zap.String("routing_key", msg.RoutingKey),
		)
		return nil // Don't retry malformed messages
	}

	logger.Debug("received event",
		zap.String("type", event.Type),
		zap.String("chat_id", event.ChatID),
		zap.String("actor_id", event.ActorID),
		zap.Int("participants", len(event.Participants)),
	)

	// Prepare the event to send to users
	userEvent := map[string]interface{}{
		"type":      event.Type,
		"timestamp": event.Timestamp,
		"actor_id":  event.ActorID,
		"chat_id":   event.ChatID,
		"data":      json.RawMessage(event.Data),
	}

	// Publish to each participant's personal channel
	for _, participantID := range event.Participants {
		if err := c.centrifugo.PublishToUser(ctx, participantID, userEvent); err != nil {
			logger.Error("failed to publish to user",
				zap.Error(err),
				zap.String("user_id", participantID),
				zap.String("event_type", event.Type),
			)
			// Continue with other participants even if one fails
		}
	}

	logger.Debug("event dispatched to participants",
		zap.String("type", event.Type),
		zap.Int("count", len(event.Participants)),
	)

	return nil
}
