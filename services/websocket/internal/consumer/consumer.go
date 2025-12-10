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
	// Configure consumer with prefetch and worker pool for high throughput
	// prefetch=100 allows RabbitMQ to deliver up to 100 unacked messages
	// workers=10 processes messages in parallel
	consumer := rabbitmq.NewConsumer(
		c.rmqConn,
		QueueName,
		ConsumerName,
		rabbitmq.WithPrefetch(100),
		rabbitmq.WithWorkers(10),
	)

	logger.Info("starting to consume messages",
		zap.String("queue", QueueName),
		zap.Int("prefetch", 100),
		zap.Int("workers", 10),
	)

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

	// Use broadcast API to send to all participants in a single HTTP request
	// This reduces HTTP calls from N (participants) to 1
	if len(event.Participants) > 0 {
		if err := c.centrifugo.BroadcastToUsers(ctx, event.Participants, userEvent); err != nil {
			logger.Error("failed to broadcast to users",
				zap.Error(err),
				zap.String("event_type", event.Type),
				zap.Int("participants", len(event.Participants)),
			)
			return err // Return error to trigger requeue
		}
	}

	logger.Debug("event broadcasted to participants",
		zap.String("type", event.Type),
		zap.Int("count", len(event.Participants)),
	)

	return nil
}
