package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/services/chat/internal/events"
	"github.com/icegreg/chat-smpl/services/chat/internal/model"
	"github.com/icegreg/chat-smpl/services/chat/internal/repository"
)

var (
	ErrAccessDenied    = errors.New("access denied")
	ErrNotParticipant  = errors.New("not a participant")
	ErrCannotWriteChat = errors.New("cannot write to this chat")
)

type ChatService interface {
	// Chat operations
	CreateChat(ctx context.Context, name string, chatType model.ChatType, createdBy uuid.UUID, participantIDs []uuid.UUID) (*model.Chat, error)
	GetChat(ctx context.Context, chatID, userID uuid.UUID) (*model.Chat, error)
	ListChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error)
	UpdateChat(ctx context.Context, chatID uuid.UUID, name string, userID uuid.UUID) (*model.Chat, error)
	DeleteChat(ctx context.Context, chatID, userID uuid.UUID) error
	SearchChats(ctx context.Context, userID uuid.UUID, query string, page, count int) ([]model.Chat, int, error)

	// Participant operations
	AddParticipant(ctx context.Context, chatID, userID, addedBy uuid.UUID, role model.ParticipantRole) (*model.ChatParticipant, error)
	RemoveParticipant(ctx context.Context, chatID, userID, removedBy uuid.UUID) error
	UpdateParticipantRole(ctx context.Context, chatID, userID, updatedBy uuid.UUID, role model.ParticipantRole) (*model.ChatParticipant, error)
	ListParticipants(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.ChatParticipant, int, error)

	// Message operations
	SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID *uuid.UUID) (*model.Message, error)
	GetMessage(ctx context.Context, messageID, userID uuid.UUID) (*model.Message, error)
	ListMessages(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Message, int, error)
	UpdateMessage(ctx context.Context, messageID, userID uuid.UUID, content string) (*model.Message, error)
	DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error
	GetThreadMessages(ctx context.Context, parentID, userID uuid.UUID, page, count int) ([]model.Message, int, error)

	// Reaction operations
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error
	ListReactions(ctx context.Context, messageID uuid.UUID) ([]model.Reaction, error)

	// Read status
	MarkAsRead(ctx context.Context, chatID, messageID, userID uuid.UUID) error
	GetReadStatus(ctx context.Context, messageID uuid.UUID) ([]uuid.UUID, int, error)

	// Favorites
	AddToFavorites(ctx context.Context, chatID, userID uuid.UUID) error
	RemoveFromFavorites(ctx context.Context, chatID, userID uuid.UUID) error

	// Archive
	ArchiveChat(ctx context.Context, chatID, userID uuid.UUID) error
	UnarchiveChat(ctx context.Context, chatID, userID uuid.UUID) error
	ListArchivedChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error)

	// Typing indicator
	SendTyping(ctx context.Context, chatID, userID uuid.UUID, isTyping bool) error
}

type chatService struct {
	repo      repository.ChatRepository
	publisher events.Publisher
}

func NewChatService(repo repository.ChatRepository, publisher events.Publisher) ChatService {
	return &chatService{
		repo:      repo,
		publisher: publisher,
	}
}

// Chat operations

func (s *chatService) CreateChat(ctx context.Context, name string, chatType model.ChatType, createdBy uuid.UUID, participantIDs []uuid.UUID) (*model.Chat, error) {
	chat := &model.Chat{
		Name:      name,
		ChatType:  chatType,
		CreatedBy: createdBy,
	}

	if err := s.repo.CreateChat(ctx, chat); err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	// Add creator as admin
	if err := s.repo.AddParticipant(ctx, &model.ChatParticipant{
		ChatID: chat.ID,
		UserID: createdBy,
		Role:   model.ParticipantRoleAdmin,
	}); err != nil {
		return nil, fmt.Errorf("failed to add creator as participant: %w", err)
	}

	// Add other participants
	for _, userID := range participantIDs {
		if userID == createdBy {
			continue
		}
		if err := s.repo.AddParticipant(ctx, &model.ChatParticipant{
			ChatID: chat.ID,
			UserID: userID,
			Role:   model.ParticipantRoleMember,
		}); err != nil {
			return nil, fmt.Errorf("failed to add participant: %w", err)
		}
	}

	// Get all participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, chat.ID)

	// Publish event
	_ = s.publisher.PublishChatCreated(ctx, chat, participants)

	return chat, nil
}

func (s *chatService) GetChat(ctx context.Context, chatID, userID uuid.UUID) (*model.Chat, error) {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	return s.repo.GetChat(ctx, chatID)
}

func (s *chatService) ListChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error) {
	return s.repo.ListChats(ctx, userID, page, count)
}

func (s *chatService) UpdateChat(ctx context.Context, chatID uuid.UUID, name string, userID uuid.UUID) (*model.Chat, error) {
	participant, err := s.repo.GetParticipant(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	if !participant.Role.CanModerate() {
		return nil, ErrAccessDenied
	}

	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return nil, err
	}

	chat.Name = name
	if err := s.repo.UpdateChat(ctx, chat); err != nil {
		return nil, err
	}

	// Publish event to all participants
	participants, _ := s.repo.GetParticipantIDs(ctx, chatID)
	_ = s.publisher.PublishChatUpdated(ctx, chat, userID, participants)

	return chat, nil
}

func (s *chatService) DeleteChat(ctx context.Context, chatID, userID uuid.UUID) error {
	participant, err := s.repo.GetParticipant(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return ErrNotParticipant
		}
		return err
	}

	if !participant.Role.CanModerate() {
		return ErrAccessDenied
	}

	// Get participants BEFORE deleting
	participants, _ := s.repo.GetParticipantIDs(ctx, chatID)

	if err := s.repo.DeleteChat(ctx, chatID); err != nil {
		return err
	}

	_ = s.publisher.PublishChatDeleted(ctx, chatID, userID, participants)

	return nil
}

func (s *chatService) SearchChats(ctx context.Context, userID uuid.UUID, query string, page, count int) ([]model.Chat, int, error) {
	return s.repo.SearchChats(ctx, userID, query, page, count)
}

// Participant operations

func (s *chatService) AddParticipant(ctx context.Context, chatID, userID, addedBy uuid.UUID, role model.ParticipantRole) (*model.ChatParticipant, error) {
	participant, err := s.repo.GetParticipant(ctx, chatID, addedBy)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	if !participant.Role.CanModerate() {
		return nil, ErrAccessDenied
	}

	newParticipant := &model.ChatParticipant{
		ChatID: chatID,
		UserID: userID,
		Role:   role,
	}

	if err := s.repo.AddParticipant(ctx, newParticipant); err != nil {
		return nil, err
	}

	return newParticipant, nil
}

func (s *chatService) RemoveParticipant(ctx context.Context, chatID, userID, removedBy uuid.UUID) error {
	if userID != removedBy {
		participant, err := s.repo.GetParticipant(ctx, chatID, removedBy)
		if err != nil {
			if errors.Is(err, repository.ErrParticipantNotFound) {
				return ErrNotParticipant
			}
			return err
		}

		if !participant.Role.CanModerate() {
			return ErrAccessDenied
		}
	}

	return s.repo.RemoveParticipant(ctx, chatID, userID)
}

func (s *chatService) UpdateParticipantRole(ctx context.Context, chatID, userID, updatedBy uuid.UUID, role model.ParticipantRole) (*model.ChatParticipant, error) {
	participant, err := s.repo.GetParticipant(ctx, chatID, updatedBy)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	if !participant.Role.CanModerate() {
		return nil, ErrAccessDenied
	}

	if err := s.repo.UpdateParticipantRole(ctx, chatID, userID, role); err != nil {
		return nil, err
	}

	return s.repo.GetParticipant(ctx, chatID, userID)
}

func (s *chatService) ListParticipants(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.ChatParticipant, int, error) {
	return s.repo.ListParticipants(ctx, chatID, page, count)
}

// Message operations

func (s *chatService) SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID *uuid.UUID) (*model.Message, error) {
	participant, err := s.repo.GetParticipant(ctx, chatID, senderID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	if !participant.Role.CanWrite() {
		return nil, ErrCannotWriteChat
	}

	message := &model.Message{
		ChatID:   chatID,
		SenderID: senderID,
		Content:  content,
		ParentID: parentID,
	}

	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Add sender info to message for event
	message.SenderUsername = participant.Username
	message.SenderDisplayName = participant.DisplayName
	message.SenderAvatarURL = participant.AvatarURL

	// Get participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, chatID)
	_ = s.publisher.PublishMessageCreated(ctx, message, participants)

	return message, nil
}

func (s *chatService) GetMessage(ctx context.Context, messageID, userID uuid.UUID) (*model.Message, error) {
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	isParticipant, err := s.repo.IsParticipant(ctx, message.ChatID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	return message, nil
}

func (s *chatService) ListMessages(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Message, int, error) {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotParticipant
	}

	return s.repo.ListMessages(ctx, chatID, page, count, nil, nil)
}

func (s *chatService) UpdateMessage(ctx context.Context, messageID, userID uuid.UUID, content string) (*model.Message, error) {
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if message.SenderID != userID {
		return nil, ErrAccessDenied
	}

	message.Content = content
	if err := s.repo.UpdateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Get participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, message.ChatID)
	_ = s.publisher.PublishMessageUpdated(ctx, message, participants)

	return message, nil
}

func (s *chatService) DeleteMessage(ctx context.Context, messageID, userID uuid.UUID) error {
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if message.SenderID != userID {
		participant, err := s.repo.GetParticipant(ctx, message.ChatID, userID)
		if err != nil {
			return err
		}
		if !participant.Role.CanModerate() {
			return ErrAccessDenied
		}
	}

	if err := s.repo.DeleteMessage(ctx, messageID); err != nil {
		return err
	}

	// Get participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, message.ChatID)
	_ = s.publisher.PublishMessageDeleted(ctx, messageID, message.ChatID, userID, participants)

	return nil
}

func (s *chatService) GetThreadMessages(ctx context.Context, parentID, userID uuid.UUID, page, count int) ([]model.Message, int, error) {
	message, err := s.repo.GetMessage(ctx, parentID)
	if err != nil {
		return nil, 0, err
	}

	isParticipant, err := s.repo.IsParticipant(ctx, message.ChatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotParticipant
	}

	return s.repo.GetThreadMessages(ctx, parentID, page, count)
}

// Reaction operations

func (s *chatService) AddReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error {
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	isParticipant, err := s.repo.IsParticipant(ctx, message.ChatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	if err := s.repo.AddReaction(ctx, &model.Reaction{
		MessageID: messageID,
		UserID:    userID,
		Reaction:  reaction,
	}); err != nil {
		return err
	}

	// Publish reaction event
	participants, _ := s.repo.GetParticipantIDs(ctx, message.ChatID)
	_ = s.publisher.PublishReactionAdded(ctx, messageID, message.ChatID, userID, reaction, participants)

	return nil
}

func (s *chatService) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error {
	message, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if err := s.repo.RemoveReaction(ctx, messageID, userID, reaction); err != nil {
		return err
	}

	// Publish reaction event
	participants, _ := s.repo.GetParticipantIDs(ctx, message.ChatID)
	_ = s.publisher.PublishReactionRemoved(ctx, messageID, message.ChatID, userID, reaction, participants)

	return nil
}

func (s *chatService) ListReactions(ctx context.Context, messageID uuid.UUID) ([]model.Reaction, error) {
	return s.repo.ListReactions(ctx, messageID)
}

// Read status

func (s *chatService) MarkAsRead(ctx context.Context, chatID, messageID, userID uuid.UUID) error {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	return s.repo.MarkAsRead(ctx, messageID, userID)
}

func (s *chatService) GetReadStatus(ctx context.Context, messageID uuid.UUID) ([]uuid.UUID, int, error) {
	readers, err := s.repo.GetReaders(ctx, messageID)
	if err != nil {
		return nil, 0, err
	}
	return readers, len(readers), nil
}

// Favorites

func (s *chatService) AddToFavorites(ctx context.Context, chatID, userID uuid.UUID) error {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	return s.repo.AddToFavorites(ctx, chatID, userID)
}

func (s *chatService) RemoveFromFavorites(ctx context.Context, chatID, userID uuid.UUID) error {
	return s.repo.RemoveFromFavorites(ctx, chatID, userID)
}

// Archive

func (s *chatService) ArchiveChat(ctx context.Context, chatID, userID uuid.UUID) error {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	return s.repo.ArchiveChat(ctx, chatID, userID)
}

func (s *chatService) UnarchiveChat(ctx context.Context, chatID, userID uuid.UUID) error {
	return s.repo.UnarchiveChat(ctx, chatID, userID)
}

func (s *chatService) ListArchivedChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error) {
	return s.repo.ListArchivedChats(ctx, userID, page, count)
}

// Typing indicator

func (s *chatService) SendTyping(ctx context.Context, chatID, userID uuid.UUID, isTyping bool) error {
	// Validate user is participant
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return ErrNotParticipant
	}

	// Get participants for event
	participants, err := s.repo.GetParticipantIDs(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	// Publish typing event
	return s.publisher.PublishTyping(ctx, chatID, userID, isTyping, participants)
}
