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

	RoutingKeyChatCreated     = "chat.created"
	RoutingKeyChatUpdated     = "chat.updated"
	RoutingKeyChatDeleted     = "chat.deleted"
	RoutingKeyMessageCreated  = "message.created"
	RoutingKeyMessageUpdated  = "message.updated"
	RoutingKeyMessageDeleted  = "message.deleted"
	RoutingKeyMessageRestored = "message.restored"
	RoutingKeyTyping          = "typing"
	RoutingKeyReactionAdded   = "reaction.added"
	RoutingKeyReactionRemoved = "reaction.removed"
	RoutingKeyThreadCreated   = "thread.created"
	RoutingKeyThreadArchived  = "thread.archived"
)

// ChatEvent is the unified event structure for websocket-service consumption
type ChatEvent struct {
	Type         string      `json:"type"`
	Timestamp    time.Time   `json:"timestamp"`
	ActorID      string      `json:"actor_id"`
	ChatID       string      `json:"chat_id"`
	Participants []string    `json:"participants"`
	Data         interface{} `json:"data"`
}

// Data payloads for events

type ChatData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ChatType  string `json:"chat_type"`
	CreatedBy string `json:"created_by"`
}

type MessageData struct {
	ID                string   `json:"id"`
	ChatID            string   `json:"chat_id"`
	SenderID          string   `json:"sender_id"`
	Content           string   `json:"content"`
	SentAt            string   `json:"sent_at"`
	UpdatedAt         *string  `json:"updated_at,omitempty"`
	ParentID          *string  `json:"parent_id,omitempty"`
	SenderUsername    *string  `json:"sender_username,omitempty"`
	SenderDisplayName *string  `json:"sender_display_name,omitempty"`
	SenderAvatarURL   *string  `json:"sender_avatar_url,omitempty"`
	FileLinkIDs       []string `json:"file_link_ids,omitempty"`
}

type MessageDeletedData struct {
	MessageID           string `json:"message_id"`
	ChatID              string `json:"chat_id"`
	IsModeratedDeletion bool   `json:"is_moderated_deletion"`
}

type MessageRestoredData struct {
	MessageID string `json:"message_id"`
	ChatID    string `json:"chat_id"`
	SenderID  string `json:"sender_id"`
	Content   string `json:"content"`
}

type TypingData struct {
	IsTyping bool   `json:"is_typing"`
	UserID   string `json:"user_id"`
}

type ReactionData struct {
	MessageID string `json:"message_id"`
	Emoji     string `json:"emoji"`
	UserID    string `json:"user_id"`
}

type ThreadData struct {
	ID                     string  `json:"id"`
	ChatID                 string  `json:"chat_id"`
	ParentMessageID        *string `json:"parent_message_id,omitempty"`
	ThreadType             string  `json:"thread_type"`
	Title                  *string `json:"title,omitempty"`
	MessageCount           int     `json:"message_count"`
	CreatedBy              *string `json:"created_by,omitempty"`
	IsArchived             bool    `json:"is_archived"`
	RestrictedParticipants bool    `json:"restricted_participants"`
}

type Publisher interface {
	PublishChatCreated(ctx context.Context, chat *model.Chat, participants []uuid.UUID) error
	PublishChatUpdated(ctx context.Context, chat *model.Chat, actorID uuid.UUID, participants []uuid.UUID) error
	PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID, participants []uuid.UUID) error
	PublishMessageCreated(ctx context.Context, message *model.Message, participants []uuid.UUID) error
	PublishMessageUpdated(ctx context.Context, message *model.Message, participants []uuid.UUID) error
	PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID, isModeratedDeletion bool, participants []uuid.UUID) error
	PublishMessageRestored(ctx context.Context, message *model.Message) error
	PublishTyping(ctx context.Context, chatID, userID uuid.UUID, isTyping bool, participants []uuid.UUID) error
	PublishReactionAdded(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error
	PublishReactionRemoved(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error
	PublishThreadCreated(ctx context.Context, thread *model.Thread, participants []uuid.UUID) error
	PublishThreadArchived(ctx context.Context, thread *model.Thread, archivedBy uuid.UUID, participants []uuid.UUID) error
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

func uuidSliceToStrings(ids []uuid.UUID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}

func (p *publisher) PublishChatCreated(ctx context.Context, chat *model.Chat, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyChatCreated,
		Timestamp:    time.Now(),
		ActorID:      chat.CreatedBy.String(),
		ChatID:       chat.ID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: ChatData{
			ID:        chat.ID.String(),
			Name:      chat.Name,
			ChatType:  string(chat.ChatType),
			CreatedBy: chat.CreatedBy.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyChatCreated, event); err != nil {
		logger.Error("failed to publish chat.created event", zap.Error(err), zap.String("chat_id", chat.ID.String()))
		return err
	}

	logger.Debug("published chat.created event", zap.String("chat_id", chat.ID.String()), zap.Int("participants", len(participants)))
	return nil
}

func (p *publisher) PublishChatUpdated(ctx context.Context, chat *model.Chat, actorID uuid.UUID, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyChatUpdated,
		Timestamp:    time.Now(),
		ActorID:      actorID.String(),
		ChatID:       chat.ID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: ChatData{
			ID:        chat.ID.String(),
			Name:      chat.Name,
			ChatType:  string(chat.ChatType),
			CreatedBy: chat.CreatedBy.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyChatUpdated, event); err != nil {
		logger.Error("failed to publish chat.updated event", zap.Error(err), zap.String("chat_id", chat.ID.String()))
		return err
	}

	logger.Debug("published chat.updated event", zap.String("chat_id", chat.ID.String()))
	return nil
}

func (p *publisher) PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyChatDeleted,
		Timestamp:    time.Now(),
		ActorID:      deletedBy.String(),
		ChatID:       chatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: map[string]string{
			"chat_id": chatID.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyChatDeleted, event); err != nil {
		logger.Error("failed to publish chat.deleted event", zap.Error(err), zap.String("chat_id", chatID.String()))
		return err
	}

	logger.Debug("published chat.deleted event", zap.String("chat_id", chatID.String()))
	return nil
}

func (p *publisher) PublishMessageCreated(ctx context.Context, message *model.Message, participants []uuid.UUID) error {
	msgData := MessageData{
		ID:       message.ID.String(),
		ChatID:   message.ChatID.String(),
		SenderID: message.SenderID.String(),
		Content:  message.Content,
		SentAt:   message.SentAt.Format(time.RFC3339),
	}
	if message.ParentID != nil {
		parentStr := message.ParentID.String()
		msgData.ParentID = &parentStr
	}
	if message.SenderUsername != nil {
		msgData.SenderUsername = message.SenderUsername
	}
	if message.SenderDisplayName != nil {
		msgData.SenderDisplayName = message.SenderDisplayName
	}
	if message.SenderAvatarURL != nil {
		msgData.SenderAvatarURL = message.SenderAvatarURL
	}
	// Add file link IDs
	for _, id := range message.FileLinkIDs {
		msgData.FileLinkIDs = append(msgData.FileLinkIDs, id.String())
	}

	event := ChatEvent{
		Type:         RoutingKeyMessageCreated,
		Timestamp:    time.Now(),
		ActorID:      message.SenderID.String(),
		ChatID:       message.ChatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data:         msgData,
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageCreated, event); err != nil {
		logger.Error("failed to publish message.created event", zap.Error(err), zap.String("message_id", message.ID.String()))
		return err
	}

	logger.Debug("published message.created event", zap.String("message_id", message.ID.String()), zap.Int("participants", len(participants)))
	return nil
}

func (p *publisher) PublishMessageUpdated(ctx context.Context, message *model.Message, participants []uuid.UUID) error {
	msgData := MessageData{
		ID:       message.ID.String(),
		ChatID:   message.ChatID.String(),
		SenderID: message.SenderID.String(),
		Content:  message.Content,
		SentAt:   message.SentAt.Format(time.RFC3339),
	}
	if message.UpdatedAt != nil {
		updatedStr := message.UpdatedAt.Format(time.RFC3339)
		msgData.UpdatedAt = &updatedStr
	}

	event := ChatEvent{
		Type:         RoutingKeyMessageUpdated,
		Timestamp:    time.Now(),
		ActorID:      message.SenderID.String(),
		ChatID:       message.ChatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data:         msgData,
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageUpdated, event); err != nil {
		logger.Error("failed to publish message.updated event", zap.Error(err), zap.String("message_id", message.ID.String()))
		return err
	}

	logger.Debug("published message.updated event", zap.String("message_id", message.ID.String()))
	return nil
}

func (p *publisher) PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID, isModeratedDeletion bool, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyMessageDeleted,
		Timestamp:    time.Now(),
		ActorID:      deletedBy.String(),
		ChatID:       chatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: MessageDeletedData{
			MessageID:           messageID.String(),
			ChatID:              chatID.String(),
			IsModeratedDeletion: isModeratedDeletion,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageDeleted, event); err != nil {
		logger.Error("failed to publish message.deleted event", zap.Error(err), zap.String("message_id", messageID.String()))
		return err
	}

	logger.Debug("published message.deleted event", zap.String("message_id", messageID.String()), zap.Bool("is_moderated", isModeratedDeletion))
	return nil
}

func (p *publisher) PublishMessageRestored(ctx context.Context, message *model.Message) error {
	// Get participants for this chat - we need repository access for this
	// For now, we'll just use the sender as the only participant in the event
	// The actual participant list should be passed from the service layer
	event := ChatEvent{
		Type:         RoutingKeyMessageRestored,
		Timestamp:    time.Now(),
		ActorID:      message.SenderID.String(),
		ChatID:       message.ChatID.String(),
		Participants: []string{}, // Will be populated by the message receiver based on chat
		Data: MessageRestoredData{
			MessageID: message.ID.String(),
			ChatID:    message.ChatID.String(),
			SenderID:  message.SenderID.String(),
			Content:   message.Content,
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyMessageRestored, event); err != nil {
		logger.Error("failed to publish message.restored event", zap.Error(err), zap.String("message_id", message.ID.String()))
		return err
	}

	logger.Debug("published message.restored event", zap.String("message_id", message.ID.String()))
	return nil
}

func (p *publisher) PublishTyping(ctx context.Context, chatID, userID uuid.UUID, isTyping bool, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyTyping,
		Timestamp:    time.Now(),
		ActorID:      userID.String(),
		ChatID:       chatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: TypingData{
			IsTyping: isTyping,
			UserID:   userID.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyTyping, event); err != nil {
		logger.Error("failed to publish typing event", zap.Error(err), zap.String("chat_id", chatID.String()))
		return err
	}

	logger.Debug("published typing event", zap.String("chat_id", chatID.String()), zap.Bool("is_typing", isTyping))
	return nil
}

func (p *publisher) PublishReactionAdded(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyReactionAdded,
		Timestamp:    time.Now(),
		ActorID:      userID.String(),
		ChatID:       chatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: ReactionData{
			MessageID: messageID.String(),
			Emoji:     emoji,
			UserID:    userID.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyReactionAdded, event); err != nil {
		logger.Error("failed to publish reaction.added event", zap.Error(err), zap.String("message_id", messageID.String()))
		return err
	}

	logger.Debug("published reaction.added event", zap.String("message_id", messageID.String()), zap.String("emoji", emoji))
	return nil
}

func (p *publisher) PublishReactionRemoved(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error {
	event := ChatEvent{
		Type:         RoutingKeyReactionRemoved,
		Timestamp:    time.Now(),
		ActorID:      userID.String(),
		ChatID:       chatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data: ReactionData{
			MessageID: messageID.String(),
			Emoji:     emoji,
			UserID:    userID.String(),
		},
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyReactionRemoved, event); err != nil {
		logger.Error("failed to publish reaction.removed event", zap.Error(err), zap.String("message_id", messageID.String()))
		return err
	}

	logger.Debug("published reaction.removed event", zap.String("message_id", messageID.String()), zap.String("emoji", emoji))
	return nil
}

func (p *publisher) PublishThreadCreated(ctx context.Context, thread *model.Thread, participants []uuid.UUID) error {
	threadData := ThreadData{
		ID:                     thread.ID.String(),
		ChatID:                 thread.ChatID.String(),
		ThreadType:             string(thread.ThreadType),
		MessageCount:           thread.MessageCount,
		IsArchived:             thread.IsArchived,
		RestrictedParticipants: thread.RestrictedParticipants,
	}
	if thread.ParentMessageID != nil {
		parentStr := thread.ParentMessageID.String()
		threadData.ParentMessageID = &parentStr
	}
	if thread.Title != nil {
		threadData.Title = thread.Title
	}
	if thread.CreatedBy != nil {
		createdByStr := thread.CreatedBy.String()
		threadData.CreatedBy = &createdByStr
	}

	actorID := ""
	if thread.CreatedBy != nil {
		actorID = thread.CreatedBy.String()
	}

	event := ChatEvent{
		Type:         RoutingKeyThreadCreated,
		Timestamp:    time.Now(),
		ActorID:      actorID,
		ChatID:       thread.ChatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data:         threadData,
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyThreadCreated, event); err != nil {
		logger.Error("failed to publish thread.created event", zap.Error(err), zap.String("thread_id", thread.ID.String()))
		return err
	}

	logger.Debug("published thread.created event", zap.String("thread_id", thread.ID.String()), zap.Int("participants", len(participants)))
	return nil
}

func (p *publisher) PublishThreadArchived(ctx context.Context, thread *model.Thread, archivedBy uuid.UUID, participants []uuid.UUID) error {
	threadData := ThreadData{
		ID:                     thread.ID.String(),
		ChatID:                 thread.ChatID.String(),
		ThreadType:             string(thread.ThreadType),
		MessageCount:           thread.MessageCount,
		IsArchived:             thread.IsArchived,
		RestrictedParticipants: thread.RestrictedParticipants,
	}
	if thread.ParentMessageID != nil {
		parentStr := thread.ParentMessageID.String()
		threadData.ParentMessageID = &parentStr
	}
	if thread.Title != nil {
		threadData.Title = thread.Title
	}
	if thread.CreatedBy != nil {
		createdByStr := thread.CreatedBy.String()
		threadData.CreatedBy = &createdByStr
	}

	event := ChatEvent{
		Type:         RoutingKeyThreadArchived,
		Timestamp:    time.Now(),
		ActorID:      archivedBy.String(),
		ChatID:       thread.ChatID.String(),
		Participants: uuidSliceToStrings(participants),
		Data:         threadData,
	}

	if err := p.rmqPublisher.Publish(ctx, RoutingKeyThreadArchived, event); err != nil {
		logger.Error("failed to publish thread.archived event", zap.Error(err), zap.String("thread_id", thread.ID.String()))
		return err
	}

	logger.Debug("published thread.archived event", zap.String("thread_id", thread.ID.String()))
	return nil
}

// NoOpPublisher is a publisher that does nothing (for testing)
type NoOpPublisher struct{}

func NewNoOpPublisher() Publisher {
	return &NoOpPublisher{}
}

func (p *NoOpPublisher) PublishChatCreated(ctx context.Context, chat *model.Chat, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishChatUpdated(ctx context.Context, chat *model.Chat, actorID uuid.UUID, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishChatDeleted(ctx context.Context, chatID, deletedBy uuid.UUID, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageCreated(ctx context.Context, message *model.Message, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageUpdated(ctx context.Context, message *model.Message, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageDeleted(ctx context.Context, messageID, chatID, deletedBy uuid.UUID, isModeratedDeletion bool, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishMessageRestored(ctx context.Context, message *model.Message) error {
	return nil
}

func (p *NoOpPublisher) PublishTyping(ctx context.Context, chatID, userID uuid.UUID, isTyping bool, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishReactionAdded(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishReactionRemoved(ctx context.Context, messageID, chatID, userID uuid.UUID, emoji string, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishThreadCreated(ctx context.Context, thread *model.Thread, participants []uuid.UUID) error {
	return nil
}

func (p *NoOpPublisher) PublishThreadArchived(ctx context.Context, thread *model.Thread, archivedBy uuid.UUID, participants []uuid.UUID) error {
	return nil
}
