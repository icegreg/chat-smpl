package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	filesPb "github.com/icegreg/chat-smpl/proto/files"
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
	SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID *uuid.UUID, fileLinkIDs, replyToIDs []uuid.UUID) (*model.Message, error)
	GetMessage(ctx context.Context, messageID, userID uuid.UUID) (*model.Message, error)
	ListMessages(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Message, int, error)
	SyncMessages(ctx context.Context, chatID, userID uuid.UUID, afterSeqNum int64, limit int) ([]model.Message, error)
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

	// Forward message
	ForwardMessage(ctx context.Context, messageID, targetChatID, senderID uuid.UUID) (*model.Message, error)

	// Thread operations
	CreateThread(ctx context.Context, chatID uuid.UUID, parentMessageID *uuid.UUID, threadType model.ThreadType, title *string, createdBy *uuid.UUID, restrictedParticipants bool) (*model.Thread, error)
	GetThread(ctx context.Context, threadID, userID uuid.UUID) (*model.Thread, error)
	ListThreads(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Thread, int, error)
	ArchiveThread(ctx context.Context, threadID, userID uuid.UUID) (*model.Thread, error)
	ListThreadMessages(ctx context.Context, threadID, userID uuid.UUID, page, count int) ([]model.Message, int, error)

	// Thread participant operations (for restricted threads)
	AddThreadParticipant(ctx context.Context, threadID, userID, addedBy uuid.UUID) error
	RemoveThreadParticipant(ctx context.Context, threadID, userID, removedBy uuid.UUID) error
	ListThreadParticipants(ctx context.Context, threadID uuid.UUID) ([]model.ThreadParticipant, error)

	// SendMessage with thread support (overloaded via optional threadID)
	SendMessageToThread(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID, threadID *uuid.UUID, fileLinkIDs, replyToIDs []uuid.UUID, isSystem bool) (*model.Message, error)

	// System thread helpers
	GetSystemThread(ctx context.Context, chatID uuid.UUID) (*model.Thread, error)
	SendSystemMessage(ctx context.Context, chatID uuid.UUID, content string) (*model.Message, error)

	// Subthread operations
	ListSubthreads(ctx context.Context, parentThreadID, userID uuid.UUID, page, count int) ([]model.Thread, int, error)
	CreateSubthread(ctx context.Context, parentThreadID uuid.UUID, title string, threadType model.ThreadType, createdBy uuid.UUID) (*model.Thread, error)

	// Chat file group operations
	InitChatFileGroups(ctx context.Context, chatID uuid.UUID) error
	GetChatFileGroups(ctx context.Context, chatID uuid.UUID) ([]model.ChatFileGroup, error)
	SyncParticipantToFileGroups(ctx context.Context, chatID, userID uuid.UUID, role model.ParticipantRole) error
	RemoveParticipantFromFileGroups(ctx context.Context, chatID, userID uuid.UUID) error
	AttachFileLinkToChat(ctx context.Context, chatID, fileLinkID, attachedBy uuid.UUID) error
	GetChatFileLinks(ctx context.Context, chatID uuid.UUID) ([]model.ChatFileLink, error)
}

type chatService struct {
	repo        repository.ChatRepository
	publisher   events.Publisher
	filesClient filesPb.FilesServiceClient
}

func NewChatService(repo repository.ChatRepository, publisher events.Publisher, filesClient filesPb.FilesServiceClient) ChatService {
	return &chatService{
		repo:        repo,
		publisher:   publisher,
		filesClient: filesClient,
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

	// Initialize file groups for the chat
	if err := s.InitChatFileGroups(ctx, chat.ID); err != nil {
		// Log error but don't fail chat creation - file groups can be initialized later
		_ = err
	}

	// Add creator as admin
	if err := s.repo.AddParticipant(ctx, &model.ChatParticipant{
		ChatID: chat.ID,
		UserID: createdBy,
		Role:   model.ParticipantRoleAdmin,
	}); err != nil {
		return nil, fmt.Errorf("failed to add creator as participant: %w", err)
	}
	// Sync creator to file groups
	_ = s.SyncParticipantToFileGroups(ctx, chat.ID, createdBy, model.ParticipantRoleAdmin)

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
		// Sync participant to file groups
		_ = s.SyncParticipantToFileGroups(ctx, chat.ID, userID, model.ParticipantRoleMember)
	}

	// Get all participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, chat.ID)

	// Publish event
	_ = s.publisher.PublishChatCreated(ctx, chat, participants)

	// Create system Activity thread for logging participant changes and events
	activityTitle := "Activity"
	activityThread := &model.Thread{
		ChatID:     chat.ID,
		ThreadType: model.ThreadTypeSystem,
		Title:      &activityTitle,
	}
	if err := s.repo.CreateThread(ctx, activityThread); err != nil {
		// Log error but don't fail chat creation
		// System thread creation is not critical
		_ = err
	}

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

	// Sync participant to file groups (this grants access to all files in the chat)
	_ = s.SyncParticipantToFileGroups(ctx, chatID, userID, role)

	// Send system message to Activity thread
	username := userID.String()[:8] // Fallback to short UUID if no username
	if newParticipant.Username != nil {
		username = *newParticipant.Username
	}
	_, _ = s.SendSystemMessage(ctx, chatID, fmt.Sprintf("%s joined the chat", username))

	return newParticipant, nil
}

func (s *chatService) RemoveParticipant(ctx context.Context, chatID, userID, removedBy uuid.UUID) error {
	// Get leaving participant info BEFORE removing (for system message)
	leavingParticipant, err := s.repo.GetParticipant(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return ErrNotParticipant
		}
		return err
	}

	if userID != removedBy {
		removerParticipant, err := s.repo.GetParticipant(ctx, chatID, removedBy)
		if err != nil {
			if errors.Is(err, repository.ErrParticipantNotFound) {
				return ErrNotParticipant
			}
			return err
		}

		if !removerParticipant.Role.CanModerate() {
			return ErrAccessDenied
		}
	}

	// Remove participant from file groups (this revokes access to all files in the chat)
	_ = s.RemoveParticipantFromFileGroups(ctx, chatID, userID)

	if err := s.repo.RemoveParticipant(ctx, chatID, userID); err != nil {
		return err
	}

	// Send system message to Activity thread
	username := userID.String()[:8] // Fallback to short UUID if no username
	if leavingParticipant.Username != nil {
		username = *leavingParticipant.Username
	}
	if userID == removedBy {
		_, _ = s.SendSystemMessage(ctx, chatID, fmt.Sprintf("%s left the chat", username))
	} else {
		_, _ = s.SendSystemMessage(ctx, chatID, fmt.Sprintf("%s was removed from the chat", username))
	}

	return nil
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

func (s *chatService) SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID *uuid.UUID, fileLinkIDs, replyToIDs []uuid.UUID) (*model.Message, error) {
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
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     content,
		ParentID:    parentID,
		FileLinkIDs: fileLinkIDs,
		ReplyToIDs:  replyToIDs,
	}

	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Attach file links to chat (adds them to file groups)
	for _, linkID := range fileLinkIDs {
		_ = s.AttachFileLinkToChat(ctx, chatID, linkID, senderID)
	}

	// Add sender info to message for event
	message.SenderUsername = participant.Username
	message.SenderDisplayName = participant.DisplayName
	message.SenderAvatarURL = participant.AvatarURL

	// Load reply_to_messages if any
	if len(replyToIDs) > 0 {
		replyMessages, err := s.repo.GetMessagesById(ctx, replyToIDs)
		if err == nil {
			message.ReplyToMessages = replyMessages
		}
	}

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

func (s *chatService) SyncMessages(ctx context.Context, chatID, userID uuid.UUID, afterSeqNum int64, limit int) ([]model.Message, error) {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	return s.repo.GetMessagesSince(ctx, chatID, afterSeqNum, limit)
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

// ForwardMessage forwards a message to another chat, creating new file links if needed
func (s *chatService) ForwardMessage(ctx context.Context, messageID, targetChatID, senderID uuid.UUID) (*model.Message, error) {
	// 1. Check sender has write access to target chat
	participant, err := s.repo.GetParticipant(ctx, targetChatID, senderID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}
	if !participant.Role.CanWrite() {
		return nil, ErrCannotWriteChat
	}

	// 2. Get original message
	original, err := s.repo.GetMessage(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original message: %w", err)
	}

	// 3. Check sender has access to original chat
	isParticipant, err := s.repo.IsParticipant(ctx, original.ChatID, senderID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	// 4. Create new file links for attached files
	var newFileLinkIDs []uuid.UUID
	if len(original.FileLinkIDs) > 0 && s.filesClient != nil {
		for _, linkID := range original.FileLinkIDs {
			// Get file_id by link_id
			fileIDResp, err := s.filesClient.GetFileIDByLinkID(ctx, &filesPb.GetFileIDByLinkIDRequest{
				LinkId: linkID.String(),
			})
			if err != nil {
				// Skip if we can't get file ID - file may have been deleted
				continue
			}

			// Create new link for the file
			newLinkResp, err := s.filesClient.CreateFileLink(ctx, &filesPb.CreateFileLinkRequest{
				FileId:    fileIDResp.FileId,
				CreatedBy: senderID.String(),
			})
			if err != nil {
				continue
			}

			newLinkID, err := uuid.Parse(newLinkResp.LinkId)
			if err != nil {
				continue
			}
			newFileLinkIDs = append(newFileLinkIDs, newLinkID)
		}
	}

	// 5. Create forwarded message
	newMessage := &model.Message{
		ChatID:                 targetChatID,
		SenderID:               senderID,
		Content:                original.Content,
		ForwardedFromMessageID: &messageID,
		ForwardedFromChatID:    &original.ChatID,
		FileLinkIDs:            newFileLinkIDs,
	}

	if err := s.repo.CreateMessage(ctx, newMessage); err != nil {
		return nil, fmt.Errorf("failed to create forwarded message: %w", err)
	}

	// 6. Attach new file links to target chat (adds them to file groups)
	for _, linkID := range newFileLinkIDs {
		_ = s.AttachFileLinkToChat(ctx, targetChatID, linkID, senderID)
	}

	// 7. Add sender info
	newMessage.SenderUsername = participant.Username
	newMessage.SenderDisplayName = participant.DisplayName
	newMessage.SenderAvatarURL = participant.AvatarURL

	// 8. Publish event
	participants, _ := s.repo.GetParticipantIDs(ctx, targetChatID)
	_ = s.publisher.PublishMessageCreated(ctx, newMessage, participants)

	return newMessage, nil
}

// Thread operations

func (s *chatService) CreateThread(ctx context.Context, chatID uuid.UUID, parentMessageID *uuid.UUID, threadType model.ThreadType, title *string, createdBy *uuid.UUID, restrictedParticipants bool) (*model.Thread, error) {
	// Validate user is participant in the chat (for user threads)
	if createdBy != nil {
		isParticipant, err := s.repo.IsParticipant(ctx, chatID, *createdBy)
		if err != nil {
			return nil, err
		}
		if !isParticipant {
			return nil, ErrNotParticipant
		}
	}

	// For reply threads, validate parent message exists and belongs to the chat
	if parentMessageID != nil {
		msg, err := s.repo.GetMessage(ctx, *parentMessageID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent message: %w", err)
		}
		if msg.ChatID != chatID {
			return nil, fmt.Errorf("parent message does not belong to this chat")
		}

		// Check if thread already exists for this parent message
		existingThread, err := s.repo.GetThreadByParentMessage(ctx, *parentMessageID)
		if err == nil && existingThread != nil {
			return existingThread, nil // Return existing thread
		}
	}

	thread := &model.Thread{
		ChatID:                 chatID,
		ParentMessageID:        parentMessageID,
		ThreadType:             threadType,
		Title:                  title,
		CreatedBy:              createdBy,
		RestrictedParticipants: restrictedParticipants,
	}

	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	// Publish thread created event
	participants, _ := s.repo.GetParticipantIDs(ctx, chatID)
	_ = s.publisher.PublishThreadCreated(ctx, thread, participants)

	return thread, nil
}

func (s *chatService) GetThread(ctx context.Context, threadID, userID uuid.UUID) (*model.Thread, error) {
	thread, err := s.repo.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}

	// Check user is participant in the chat
	isParticipant, err := s.repo.IsParticipant(ctx, thread.ChatID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	// For restricted threads, check thread participation
	if thread.RestrictedParticipants {
		isThreadParticipant, err := s.repo.IsThreadParticipant(ctx, threadID, userID)
		if err != nil {
			return nil, err
		}
		if !isThreadParticipant {
			return nil, ErrAccessDenied
		}
	}

	return thread, nil
}

func (s *chatService) ListThreads(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Thread, int, error) {
	isParticipant, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotParticipant
	}

	return s.repo.ListThreads(ctx, chatID, page, count)
}

func (s *chatService) ArchiveThread(ctx context.Context, threadID, userID uuid.UUID) (*model.Thread, error) {
	thread, err := s.repo.GetThread(ctx, threadID)
	if err != nil {
		return nil, err
	}

	// Check user has moderator permissions
	participant, err := s.repo.GetParticipant(ctx, thread.ChatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, ErrNotParticipant
		}
		return nil, err
	}

	// Thread creator or moderators can archive
	if thread.CreatedBy != nil && *thread.CreatedBy != userID && !participant.Role.CanModerate() {
		return nil, ErrAccessDenied
	}

	if err := s.repo.ArchiveThread(ctx, threadID); err != nil {
		return nil, err
	}

	thread.IsArchived = true

	// Publish thread archived event
	participants, _ := s.repo.GetParticipantIDs(ctx, thread.ChatID)
	_ = s.publisher.PublishThreadArchived(ctx, thread, userID, participants)

	return thread, nil
}

func (s *chatService) ListThreadMessages(ctx context.Context, threadID, userID uuid.UUID, page, count int) ([]model.Message, int, error) {
	thread, err := s.repo.GetThread(ctx, threadID)
	if err != nil {
		return nil, 0, err
	}

	// Check user is participant in the chat
	isParticipant, err := s.repo.IsParticipant(ctx, thread.ChatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotParticipant
	}

	// For restricted threads, check thread participation
	if thread.RestrictedParticipants {
		isThreadParticipant, err := s.repo.IsThreadParticipant(ctx, threadID, userID)
		if err != nil {
			return nil, 0, err
		}
		if !isThreadParticipant {
			return nil, 0, ErrAccessDenied
		}
	}

	return s.repo.ListThreadMessages(ctx, threadID, page, count)
}

// Thread participant operations

func (s *chatService) AddThreadParticipant(ctx context.Context, threadID, userID, addedBy uuid.UUID) error {
	thread, err := s.repo.GetThread(ctx, threadID)
	if err != nil {
		return err
	}

	// Check addedBy has moderator permissions
	participant, err := s.repo.GetParticipant(ctx, thread.ChatID, addedBy)
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return ErrNotParticipant
		}
		return err
	}

	// Thread creator or moderators can add participants
	if thread.CreatedBy != nil && *thread.CreatedBy != addedBy && !participant.Role.CanModerate() {
		return ErrAccessDenied
	}

	// User must be participant in the chat
	isParticipant, err := s.repo.IsParticipant(ctx, thread.ChatID, userID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return fmt.Errorf("user is not a participant in the chat")
	}

	threadParticipant := &model.ThreadParticipant{
		ThreadID: threadID,
		UserID:   userID,
	}

	return s.repo.AddThreadParticipant(ctx, threadParticipant)
}

func (s *chatService) RemoveThreadParticipant(ctx context.Context, threadID, userID, removedBy uuid.UUID) error {
	thread, err := s.repo.GetThread(ctx, threadID)
	if err != nil {
		return err
	}

	// Users can remove themselves, or moderators/creators can remove others
	if userID != removedBy {
		participant, err := s.repo.GetParticipant(ctx, thread.ChatID, removedBy)
		if err != nil {
			if errors.Is(err, repository.ErrParticipantNotFound) {
				return ErrNotParticipant
			}
			return err
		}

		if thread.CreatedBy != nil && *thread.CreatedBy != removedBy && !participant.Role.CanModerate() {
			return ErrAccessDenied
		}
	}

	return s.repo.RemoveThreadParticipant(ctx, threadID, userID)
}

func (s *chatService) ListThreadParticipants(ctx context.Context, threadID uuid.UUID) ([]model.ThreadParticipant, error) {
	return s.repo.ListThreadParticipants(ctx, threadID)
}

// SendMessageToThread sends a message with optional thread support
func (s *chatService) SendMessageToThread(ctx context.Context, chatID, senderID uuid.UUID, content string, parentID, threadID *uuid.UUID, fileLinkIDs, replyToIDs []uuid.UUID, isSystem bool) (*model.Message, error) {
	var participant *model.ChatParticipant
	var err error

	// System messages don't require participant check
	if !isSystem {
		participant, err = s.repo.GetParticipant(ctx, chatID, senderID)
		if err != nil {
			if errors.Is(err, repository.ErrParticipantNotFound) {
				return nil, ErrNotParticipant
			}
			return nil, err
		}

		if !participant.Role.CanWrite() {
			return nil, ErrCannotWriteChat
		}
	}

	// If threadID provided, validate it belongs to this chat
	if threadID != nil {
		thread, err := s.repo.GetThread(ctx, *threadID)
		if err != nil {
			return nil, fmt.Errorf("failed to get thread: %w", err)
		}
		if thread.ChatID != chatID {
			return nil, fmt.Errorf("thread does not belong to this chat")
		}

		// For restricted threads, check thread participation (unless system message)
		if thread.RestrictedParticipants && !isSystem {
			isThreadParticipant, err := s.repo.IsThreadParticipant(ctx, *threadID, senderID)
			if err != nil {
				return nil, err
			}
			if !isThreadParticipant {
				return nil, ErrAccessDenied
			}
		}
	}

	message := &model.Message{
		ChatID:      chatID,
		SenderID:    senderID,
		Content:     content,
		ParentID:    parentID,
		ThreadID:    threadID,
		FileLinkIDs: fileLinkIDs,
		ReplyToIDs:  replyToIDs,
		IsSystem:    isSystem,
	}

	if err := s.repo.CreateMessage(ctx, message); err != nil {
		return nil, err
	}

	// Attach file links to chat (adds them to file groups)
	for _, linkID := range fileLinkIDs {
		_ = s.AttachFileLinkToChat(ctx, chatID, linkID, senderID)
	}

	// Add sender info to message for event (for non-system messages)
	if participant != nil {
		message.SenderUsername = participant.Username
		message.SenderDisplayName = participant.DisplayName
		message.SenderAvatarURL = participant.AvatarURL
	}

	// Load reply_to_messages if any
	if len(replyToIDs) > 0 {
		replyMessages, err := s.repo.GetMessagesById(ctx, replyToIDs)
		if err == nil {
			message.ReplyToMessages = replyMessages
		}
	}

	// Get participants for event
	participants, _ := s.repo.GetParticipantIDs(ctx, chatID)
	_ = s.publisher.PublishMessageCreated(ctx, message, participants)

	return message, nil
}

// System thread helpers

func (s *chatService) GetSystemThread(ctx context.Context, chatID uuid.UUID) (*model.Thread, error) {
	return s.repo.GetSystemThread(ctx, chatID)
}

func (s *chatService) SendSystemMessage(ctx context.Context, chatID uuid.UUID, content string) (*model.Message, error) {
	// Get system thread
	systemThread, err := s.repo.GetSystemThread(ctx, chatID)
	if err != nil {
		// If no system thread exists, create one
		if errors.Is(err, repository.ErrThreadNotFound) {
			activityTitle := "Activity"
			systemThread, err = s.CreateThread(ctx, chatID, nil, model.ThreadTypeSystem, &activityTitle, nil, false)
			if err != nil {
				return nil, fmt.Errorf("failed to create system thread: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get system thread: %w", err)
		}
	}

	// Send message to system thread
	return s.SendMessageToThread(ctx, chatID, uuid.Nil, content, nil, &systemThread.ID, nil, nil, true)
}

// Subthread operations

func (s *chatService) ListSubthreads(ctx context.Context, parentThreadID, userID uuid.UUID, page, count int) ([]model.Thread, int, error) {
	// Get the parent thread to verify access
	parentThread, err := s.repo.GetThread(ctx, parentThreadID)
	if err != nil {
		return nil, 0, err
	}

	// Check user is participant in the chat
	isParticipant, err := s.repo.IsParticipant(ctx, parentThread.ChatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotParticipant
	}

	// Use cascading permission check via repository
	return s.repo.ListSubthreads(ctx, parentThreadID, userID, page, count)
}

func (s *chatService) CreateSubthread(ctx context.Context, parentThreadID uuid.UUID, title string, threadType model.ThreadType, createdBy uuid.UUID) (*model.Thread, error) {
	// Get the parent thread
	parentThread, err := s.repo.GetThread(ctx, parentThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent thread: %w", err)
	}

	// Check user is participant in the chat
	isParticipant, err := s.repo.IsParticipant(ctx, parentThread.ChatID, createdBy)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}

	// Check user has access to parent thread (cascading permission)
	hasAccess, err := s.repo.HasThreadAccess(ctx, parentThreadID, createdBy)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrAccessDenied
	}

	// Check depth limit (max 5 levels of nesting)
	if parentThread.Depth >= 5 {
		return nil, fmt.Errorf("maximum subthread depth exceeded (max 5)")
	}

	// Create the subthread
	thread := &model.Thread{
		ChatID:                 parentThread.ChatID,
		ParentThreadID:         &parentThreadID,
		ThreadType:             threadType,
		Title:                  &title,
		CreatedBy:              &createdBy,
		RestrictedParticipants: false, // Inherit from parent by default
	}

	if err := s.repo.CreateThread(ctx, thread); err != nil {
		return nil, fmt.Errorf("failed to create subthread: %w", err)
	}

	// Re-fetch to get depth set by database trigger
	thread, err = s.repo.GetThread(ctx, thread.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created subthread: %w", err)
	}

	// Publish thread created event
	participants, _ := s.repo.GetParticipantIDs(ctx, parentThread.ChatID)
	_ = s.publisher.PublishThreadCreated(ctx, thread, participants)

	return thread, nil
}

// Chat file group operations

// InitChatFileGroups creates file groups for a chat in the Files Service
// Called when a new chat is created
func (s *chatService) InitChatFileGroups(ctx context.Context, chatID uuid.UUID) error {
	if s.filesClient == nil {
		return nil // No files client, skip
	}

	chat, err := s.repo.GetChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get chat: %w", err)
	}

	// Create "moderate" group (can_read, can_delete, can_transfer) for admins
	moderateResp, err := s.filesClient.CreateFileGroup(ctx, &filesPb.CreateFileGroupRequest{
		Name:        fmt.Sprintf("chat_%s_moderate", chatID.String()),
		CanRead:     true,
		CanDelete:   true,
		CanTransfer: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create moderate file group: %w", err)
	}

	moderateGroupID, err := uuid.Parse(moderateResp.Group.Id)
	if err != nil {
		return fmt.Errorf("failed to parse moderate group ID: %w", err)
	}

	// Save mapping in chat_file_groups
	if err := s.repo.CreateChatFileGroup(ctx, &model.ChatFileGroup{
		ChatID:    chatID,
		GroupID:   moderateGroupID,
		GroupType: model.ChatFileGroupTypeModerate,
	}); err != nil {
		return fmt.Errorf("failed to save moderate file group mapping: %w", err)
	}

	// Create "read" group (can_read only) for regular members
	readResp, err := s.filesClient.CreateFileGroup(ctx, &filesPb.CreateFileGroupRequest{
		Name:        fmt.Sprintf("chat_%s_read", chat.ID.String()),
		CanRead:     true,
		CanDelete:   false,
		CanTransfer: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create read file group: %w", err)
	}

	readGroupID, err := uuid.Parse(readResp.Group.Id)
	if err != nil {
		return fmt.Errorf("failed to parse read group ID: %w", err)
	}

	// Save mapping in chat_file_groups
	if err := s.repo.CreateChatFileGroup(ctx, &model.ChatFileGroup{
		ChatID:    chatID,
		GroupID:   readGroupID,
		GroupType: model.ChatFileGroupTypeRead,
	}); err != nil {
		return fmt.Errorf("failed to save read file group mapping: %w", err)
	}

	return nil
}

// GetChatFileGroups returns file group mappings for a chat
func (s *chatService) GetChatFileGroups(ctx context.Context, chatID uuid.UUID) ([]model.ChatFileGroup, error) {
	return s.repo.GetChatFileGroups(ctx, chatID)
}

// SyncParticipantToFileGroups adds a participant to appropriate file groups based on role
func (s *chatService) SyncParticipantToFileGroups(ctx context.Context, chatID, userID uuid.UUID, role model.ParticipantRole) error {
	if s.filesClient == nil {
		return nil
	}

	groups, err := s.repo.GetChatFileGroups(ctx, chatID)
	if err != nil {
		// If no groups exist yet, initialize them
		if errors.Is(err, repository.ErrChatFileGroupNotFound) {
			if initErr := s.InitChatFileGroups(ctx, chatID); initErr != nil {
				return initErr
			}
			groups, err = s.repo.GetChatFileGroups(ctx, chatID)
			if err != nil {
				return fmt.Errorf("failed to get file groups after init: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get file groups: %w", err)
		}
	}

	// Determine which group to add user to based on role
	for _, group := range groups {
		shouldBeInGroup := false

		switch group.GroupType {
		case model.ChatFileGroupTypeModerate:
			// Only admins in moderate group
			shouldBeInGroup = role.CanModerate()
		case model.ChatFileGroupTypeRead:
			// All members (including admins) in read group
			shouldBeInGroup = true
		}

		if shouldBeInGroup {
			_, err := s.filesClient.AddUserToGroup(ctx, &filesPb.AddUserToGroupRequest{
				GroupId: group.GroupID.String(),
				UserId:  userID.String(),
			})
			if err != nil {
				// Log but don't fail - user might already be in group
				_ = err
			}
		}
	}

	return nil
}

// RemoveParticipantFromFileGroups removes a participant from all file groups of a chat
func (s *chatService) RemoveParticipantFromFileGroups(ctx context.Context, chatID, userID uuid.UUID) error {
	if s.filesClient == nil {
		return nil
	}

	groups, err := s.repo.GetChatFileGroups(ctx, chatID)
	if err != nil {
		if errors.Is(err, repository.ErrChatFileGroupNotFound) {
			return nil // No groups, nothing to do
		}
		return fmt.Errorf("failed to get file groups: %w", err)
	}

	// Collect group IDs
	groupIDs := make([]string, len(groups))
	for i, group := range groups {
		groupIDs[i] = group.GroupID.String()
	}

	// Remove user from all groups and revoke permissions
	_, err = s.filesClient.RemoveUserFromAllGroupFiles(ctx, &filesPb.RemoveUserFromAllGroupFilesRequest{
		GroupIds: groupIDs,
		UserId:   userID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to remove user from file groups: %w", err)
	}

	return nil
}

// AttachFileLinkToChat tracks a file link as attached to a chat and adds it to chat file groups
func (s *chatService) AttachFileLinkToChat(ctx context.Context, chatID, fileLinkID, attachedBy uuid.UUID) error {
	// Save the link to chat_file_links
	if err := s.repo.CreateChatFileLink(ctx, &model.ChatFileLink{
		ChatID:     chatID,
		FileLinkID: fileLinkID,
		AttachedBy: attachedBy,
	}); err != nil {
		return fmt.Errorf("failed to create chat file link: %w", err)
	}

	// Add the file link to all chat file groups
	if s.filesClient != nil {
		groups, err := s.repo.GetChatFileGroups(ctx, chatID)
		if err == nil && len(groups) > 0 {
			groupIDs := make([]string, len(groups))
			for i, g := range groups {
				groupIDs[i] = g.GroupID.String()
			}
			_, _ = s.filesClient.AddFileLinkToGroups(ctx, &filesPb.AddFileLinkToGroupsRequest{
				FileLinkId: fileLinkID.String(),
				GroupIds:   groupIDs,
			})
		}
	}

	return nil
}

// GetChatFileLinks returns all file links attached to a chat
func (s *chatService) GetChatFileLinks(ctx context.Context, chatID uuid.UUID) ([]model.ChatFileLink, error) {
	return s.repo.GetChatFileLinks(ctx, chatID)
}
