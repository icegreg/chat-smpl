package events

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	"github.com/icegreg/chat-smpl/services/chat/internal/model"
	"go.uber.org/zap"
)

const (
	ExchangeName = "chat.events"

	RoutingKeyChatCreated    = "chat.created"
	RoutingKeyChatDeleted    = "chat.deleted"
	RoutingKeyMessageCreated = "message.created"
	RoutingKeyMessageUpdated = "message.updated"
	RoutingKeyMessageDeleted = "message.deleted"
)

type ChatCreatedEvent struct {
	ChatID    uuid.UUID `json:"chat_id"`
	Name      string    `json:"name"`
	ChatType  string    `json:"chat_type"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type ChatDeletedEvent struct {
	ChatID    uuid.UUID `json:"chat_id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type MessageCreatedEvent struct {
	MessageID uuid.UUID  `json:"message_id"`
	ChatID    uuid.UUID  `json:"chat_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	SenderID  uuid.UUID  `json:"sender_id"`
	Content   string     `json:"content"`
	SentAt    time.Time  `json:"sent_at"`
}

type MessageUpdatedEvent struct {
	MessageID uuid.UUID `json:"message_id"`
	ChatID    uuid.UUID `json:"chat_id"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MessageDeletedEvent struct {
	MessageID uuid.UUID `json:"message_id"`
	ChatID    uuid.UUID `json:"chat_id"`
	DeletedBy uuid.UUID `json:"deleted_by"`
}

type Publisher interface {
	PublishChatCreated(ctx context.Context, chat *model.Chat) error
	PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID) error
	PublishMessageCreated(ctx context.Context, message *model.Message) error
	PublishMessageUpdated(ctx context.Context, message *model.Message) error
	PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID) error
}

type publisher struct {
	rmqPublisher *rabbitmq.Publisher
}

func NewPublisher(conn *rabbitmq.Connection) (Publisher, error) {
	// Declare exchange
	err := conn.DeclareExchange(rabbitmq.Exchange{
		Name:       ExchangeName,
		Kind:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	})
	if err != nil {
		return nil, err
	}

	return &publisher{
		rmqPublisher: rabbitmq.NewPublisher(conn, ExchangeName),
	}, nil
}

func (p *publisher) PublishChatCreated(ctx context.Context, chat *model.Chat) error {
	event := rabbitmq.Event{
		Type:      RoutingKeyChatCreated,
		Timestamp: time.Now(),
		Payload: ChatCreatedEvent{
			ChatID:    chat.ID,
			Name:      chat.Name,
			ChatType:  string(chat.ChatType),
			CreatedBy: chat.CreatedBy,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyChatCreated, event); err != nil {
		logger.Error("failed to publish chat.created event", zap.Error(err), zap.String("chat_id", chat.ID.String()))
		return err
	}

	logger.Debug("published chat.created event", zap.String("chat_id", chat.ID.String()))
	return nil
}

func (p *publisher) PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID) error {
	event := rabbitmq.Event{
		Type:      RoutingKeyChatDeleted,
		Timestamp: time.Now(),
		Payload: ChatDeletedEvent{
			ChatID:    chatID,
			DeletedBy: deletedBy,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyChatDeleted, event); err != nil {
		logger.Error("failed to publish chat.deleted event", zap.Error(err), zap.String("chat_id", chatID.String()))
		return err
	}

	logger.Debug("published chat.deleted event", zap.String("chat_id", chatID.String()))
	return nil
}

func (p *publisher) PublishMessageCreated(ctx context.Context, message *model.Message) error {
	event := rabbitmq.Event{
		Type:      RoutingKeyMessageCreated,
		Timestamp: time.Now(),
		Payload: MessageCreatedEvent{
			MessageID: message.ID,
			ChatID:    message.ChatID,
			ParentID:  message.ParentID,
			SenderID:  message.SenderID,
			Content:   message.Content,
			SentAt:    message.SentAt,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageCreated, event); err != nil {
		logger.Error("failed to publish message.created event", zap.Error(err), zap.String("message_id", message.ID.String()))
		return err
	}

	logger.Debug("published message.created event", zap.String("message_id", message.ID.String()))
	return nil
}

func (p *publisher) PublishMessageUpdated(ctx context.Context, message *model.Message) error {
	updatedAt := time.Now()
	if message.UpdatedAt != nil {
		updatedAt = *message.UpdatedAt
	}

	event := rabbitmq.Event{
		Type:      RoutingKeyMessageUpdated,
		Timestamp: time.Now(),
		Payload: MessageUpdatedEvent{
			MessageID: message.ID,
			ChatID:    message.ChatID,
			Content:   message.Content,
			UpdatedAt: updatedAt,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageUpdated, event); err != nil {
		logger.Error("failed to publish message.updated event", zap.Error(err), zap.String("message_id", message.ID.String()))
		return err
	}

	logger.Debug("published message.updated event", zap.String("message_id", message.ID.String()))
	return nil
}

func (p *publisher) PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID) error {
	event := rabbitmq.Event{
		Type:      RoutingKeyMessageDeleted,
		Timestamp: time.Now(),
		Payload: MessageDeletedEvent{
			MessageID: messageID,
			ChatID:    chatID,
			DeletedBy: deletedBy,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageDeleted, event); err != nil {
		logger.Error("failed to publish message.deleted event", zap.Error(err), zap.String("message_id", messageID.String()))
		return err
	}

	logger.Debug("published message.deleted event", zap.String("message_id", messageID.String()))
	return nil
}

// NoOpPublisher is a publisher that does nothing (for testing)
type NoOpPublisher struct{}

func NewNoOpPublisher() Publisher {
	return &NoOpPublisher{}
}

func (p *NoOpPublisher) PublishChatCreated(ctx context.Context, chat *model.Chat) error {
	return nil
}

func (p *NoOpPublisher) PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageCreated(ctx context.Context, message *model.Message) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageUpdated(ctx context.Context, message *model.Message) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID) error {
	return nil
}
