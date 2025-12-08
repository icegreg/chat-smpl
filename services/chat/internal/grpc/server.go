package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/icegreg/chat-smpl/proto/chat"
	"github.com/icegreg/chat-smpl/services/chat/internal/model"
	"github.com/icegreg/chat-smpl/services/chat/internal/repository"
	"github.com/icegreg/chat-smpl/services/chat/internal/service"
)

// ChatServer implements the gRPC ChatService interface
type ChatServer struct {
	pb.UnimplementedChatServiceServer
	chatService service.ChatService
}

func NewChatServer(chatService service.ChatService) *ChatServer {
	return &ChatServer{
		chatService: chatService,
	}
}

// Helper functions

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func parseUUIDPtr(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, repository.ErrChatNotFound):
		return status.Error(codes.NotFound, "chat not found")
	case errors.Is(err, repository.ErrMessageNotFound):
		return status.Error(codes.NotFound, "message not found")
	case errors.Is(err, repository.ErrParticipantNotFound):
		return status.Error(codes.NotFound, "participant not found")
	case errors.Is(err, service.ErrNotParticipant):
		return status.Error(codes.PermissionDenied, "not a participant")
	case errors.Is(err, service.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, "access denied")
	case errors.Is(err, service.ErrCannotWriteChat):
		return status.Error(codes.PermissionDenied, "cannot write to this chat")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func toChatType(ct pb.ChatType) model.ChatType {
	switch ct {
	case pb.ChatType_CHAT_TYPE_PRIVATE:
		return model.ChatTypePrivate
	case pb.ChatType_CHAT_TYPE_GROUP:
		return model.ChatTypeGroup
	case pb.ChatType_CHAT_TYPE_CHANNEL:
		return model.ChatTypeChannel
	default:
		return model.ChatTypeGroup
	}
}

func toProtoChatType(ct model.ChatType) pb.ChatType {
	switch ct {
	case model.ChatTypePrivate:
		return pb.ChatType_CHAT_TYPE_PRIVATE
	case model.ChatTypeGroup:
		return pb.ChatType_CHAT_TYPE_GROUP
	case model.ChatTypeChannel:
		return pb.ChatType_CHAT_TYPE_CHANNEL
	default:
		return pb.ChatType_CHAT_TYPE_GROUP
	}
}

func toParticipantRole(role pb.ParticipantRole) model.ParticipantRole {
	switch role {
	case pb.ParticipantRole_PARTICIPANT_ROLE_ADMIN:
		return model.ParticipantRoleAdmin
	case pb.ParticipantRole_PARTICIPANT_ROLE_MEMBER:
		return model.ParticipantRoleMember
	case pb.ParticipantRole_PARTICIPANT_ROLE_READONLY:
		return model.ParticipantRoleReadonly
	default:
		return model.ParticipantRoleMember
	}
}

func toProtoParticipantRole(role model.ParticipantRole) pb.ParticipantRole {
	switch role {
	case model.ParticipantRoleAdmin:
		return pb.ParticipantRole_PARTICIPANT_ROLE_ADMIN
	case model.ParticipantRoleMember:
		return pb.ParticipantRole_PARTICIPANT_ROLE_MEMBER
	case model.ParticipantRoleReadonly:
		return pb.ParticipantRole_PARTICIPANT_ROLE_READONLY
	default:
		return pb.ParticipantRole_PARTICIPANT_ROLE_MEMBER
	}
}

func chatToProto(c *model.Chat) *pb.Chat {
	if c == nil {
		return nil
	}
	return &pb.Chat{
		Id:        c.ID.String(),
		Name:      c.Name,
		ChatType:  toProtoChatType(c.ChatType),
		CreatedBy: c.CreatedBy.String(),
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
}

func messageToProto(m *model.Message) *pb.Message {
	if m == nil {
		return nil
	}
	msg := &pb.Message{
		Id:        m.ID.String(),
		ChatId:    m.ChatID.String(),
		SenderId:  m.SenderID.String(),
		Content:   m.Content,
		SentAt:    timestamppb.New(m.SentAt),
		IsDeleted: m.IsDeleted,
		SeqNum:    m.SeqNum,
	}
	if m.ParentID != nil {
		msg.ParentId = m.ParentID.String()
	}
	if m.UpdatedAt != nil {
		msg.UpdatedAt = timestamppb.New(*m.UpdatedAt)
	}
	if m.SenderUsername != nil {
		msg.SenderUsername = *m.SenderUsername
	}
	if m.SenderDisplayName != nil {
		msg.SenderDisplayName = *m.SenderDisplayName
	}
	if m.SenderAvatarURL != nil {
		msg.SenderAvatarUrl = *m.SenderAvatarURL
	}
	// Add file link IDs
	for _, id := range m.FileLinkIDs {
		msg.FileLinkIds = append(msg.FileLinkIds, id.String())
	}
	// Add forwarded message fields
	if m.ForwardedFromMessageID != nil {
		msg.ForwardedFromMessageId = m.ForwardedFromMessageID.String()
	}
	if m.ForwardedFromChatID != nil {
		msg.ForwardedFromChatId = m.ForwardedFromChatID.String()
	}
	return msg
}

func participantToProto(p *model.ChatParticipant) *pb.ChatParticipant {
	if p == nil {
		return nil
	}
	result := &pb.ChatParticipant{
		Id:       p.ID.String(),
		ChatId:   p.ChatID.String(),
		UserId:   p.UserID.String(),
		Role:     toProtoParticipantRole(p.Role),
		JoinedAt: timestamppb.New(p.JoinedAt),
	}
	if p.Username != nil {
		result.Username = *p.Username
	}
	if p.Email != nil {
		result.Email = *p.Email
	}
	if p.DisplayName != nil {
		result.DisplayName = *p.DisplayName
	}
	if p.AvatarURL != nil {
		result.AvatarUrl = *p.AvatarURL
	}
	return result
}

// Chat operations

func (s *ChatServer) CreateChat(ctx context.Context, req *pb.CreateChatRequest) (*pb.Chat, error) {
	createdByID, err := parseUUID(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by ID")
	}

	var participants []uuid.UUID
	for _, p := range req.ParticipantIds {
		id, err := parseUUID(p)
		if err != nil {
			continue
		}
		participants = append(participants, id)
	}

	chat, err := s.chatService.CreateChat(ctx, req.Name, toChatType(req.ChatType), createdByID, participants)
	if err != nil {
		return nil, handleError(err)
	}

	return chatToProto(chat), nil
}

func (s *ChatServer) GetChat(ctx context.Context, req *pb.GetChatRequest) (*pb.Chat, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	chat, err := s.chatService.GetChat(ctx, chatID, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return chatToProto(chat), nil
}

func (s *ChatServer) ListChats(ctx context.Context, req *pb.ListChatsRequest) (*pb.ListChatsResponse, error) {
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 20
	}

	chats, total, err := s.chatService.ListChats(ctx, userID, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoChats := make([]*pb.Chat, len(chats))
	for i, c := range chats {
		protoChats[i] = chatToProto(&c)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListChatsResponse{
		Chats: protoChats,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

func (s *ChatServer) UpdateChat(ctx context.Context, req *pb.UpdateChatRequest) (*pb.Chat, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	chat, err := s.chatService.UpdateChat(ctx, chatID, req.Name, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return chatToProto(chat), nil
}

func (s *ChatServer) DeleteChat(ctx context.Context, req *pb.DeleteChatRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.DeleteChat(ctx, chatID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) SearchChats(ctx context.Context, req *pb.SearchChatsRequest) (*pb.ListChatsResponse, error) {
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 20
	}

	chats, total, err := s.chatService.SearchChats(ctx, userID, req.Query, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoChats := make([]*pb.Chat, len(chats))
	for i, c := range chats {
		protoChats[i] = chatToProto(&c)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListChatsResponse{
		Chats: protoChats,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

// Participant operations

func (s *ChatServer) AddParticipant(ctx context.Context, req *pb.AddParticipantRequest) (*pb.ChatParticipant, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	addedByID, err := parseUUID(req.AddedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid added_by")
	}

	participant, err := s.chatService.AddParticipant(ctx, chatID, userID, addedByID, toParticipantRole(req.Role))
	if err != nil {
		return nil, handleError(err)
	}

	return participantToProto(participant), nil
}

func (s *ChatServer) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	removedByID, err := parseUUID(req.RemovedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid removed_by")
	}

	if err := s.chatService.RemoveParticipant(ctx, chatID, userID, removedByID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) UpdateParticipantRole(ctx context.Context, req *pb.UpdateParticipantRoleRequest) (*pb.ChatParticipant, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	updatedByID, err := parseUUID(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	participant, err := s.chatService.UpdateParticipantRole(ctx, chatID, userID, updatedByID, toParticipantRole(req.Role))
	if err != nil {
		return nil, handleError(err)
	}

	return participantToProto(participant), nil
}

func (s *ChatServer) ListParticipants(ctx context.Context, req *pb.ListParticipantsRequest) (*pb.ListParticipantsResponse, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 20
	}

	participants, total, err := s.chatService.ListParticipants(ctx, chatID, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoParticipants := make([]*pb.ChatParticipant, len(participants))
	for i, p := range participants {
		protoParticipants[i] = participantToProto(&p)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListParticipantsResponse{
		Participants: protoParticipants,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

// Message operations

func (s *ChatServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.Message, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	senderID, err := parseUUID(req.SenderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sender_id")
	}

	// Parse file link IDs
	var fileLinkIDs []uuid.UUID
	for _, id := range req.FileLinkIds {
		fileLinkID, err := parseUUID(id)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid file_link_id: "+id)
		}
		fileLinkIDs = append(fileLinkIDs, fileLinkID)
	}

	message, err := s.chatService.SendMessage(ctx, chatID, senderID, req.Content, parseUUIDPtr(req.ParentId), fileLinkIDs)
	if err != nil {
		return nil, handleError(err)
	}

	return messageToProto(message), nil
}

func (s *ChatServer) GetMessage(ctx context.Context, req *pb.GetMessageRequest) (*pb.Message, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	message, err := s.chatService.GetMessage(ctx, messageID, userID)
	if err != nil {
		return nil, handleError(err)
	}

	return messageToProto(message), nil
}

func (s *ChatServer) ListMessages(ctx context.Context, req *pb.ListMessagesRequest) (*pb.ListMessagesResponse, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 50
	}

	messages, total, err := s.chatService.ListMessages(ctx, chatID, userID, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoMessages := make([]*pb.Message, len(messages))
	for i, m := range messages {
		protoMessages[i] = messageToProto(&m)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListMessagesResponse{
		Messages: protoMessages,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

func (s *ChatServer) SyncMessages(ctx context.Context, req *pb.SyncMessagesRequest) (*pb.SyncMessagesResponse, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	limit := int(req.Limit)
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	messages, err := s.chatService.SyncMessages(ctx, chatID, userID, req.AfterSeqNum, limit)
	if err != nil {
		return nil, handleError(err)
	}

	protoMessages := make([]*pb.Message, len(messages))
	for i, m := range messages {
		protoMessages[i] = messageToProto(&m)
	}

	// hasMore is true if we got exactly limit messages (might be more)
	hasMore := len(messages) == limit

	return &pb.SyncMessagesResponse{
		Messages: protoMessages,
		HasMore:  hasMore,
	}, nil
}

func (s *ChatServer) UpdateMessage(ctx context.Context, req *pb.UpdateMessageRequest) (*pb.Message, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	message, err := s.chatService.UpdateMessage(ctx, messageID, userID, req.Content)
	if err != nil {
		return nil, handleError(err)
	}

	return messageToProto(message), nil
}

func (s *ChatServer) DeleteMessage(ctx context.Context, req *pb.DeleteMessageRequest) (*emptypb.Empty, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.DeleteMessage(ctx, messageID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) GetThreadMessages(ctx context.Context, req *pb.GetThreadMessagesRequest) (*pb.ListMessagesResponse, error) {
	parentID, err := parseUUID(req.ParentId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 50
	}

	messages, total, err := s.chatService.GetThreadMessages(ctx, parentID, userID, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoMessages := make([]*pb.Message, len(messages))
	for i, m := range messages {
		protoMessages[i] = messageToProto(&m)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListMessagesResponse{
		Messages: protoMessages,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

// Reaction operations

func (s *ChatServer) AddReaction(ctx context.Context, req *pb.AddReactionRequest) (*emptypb.Empty, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.AddReaction(ctx, messageID, userID, req.Reaction); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) RemoveReaction(ctx context.Context, req *pb.RemoveReactionRequest) (*emptypb.Empty, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.RemoveReaction(ctx, messageID, userID, req.Reaction); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) ListReactions(ctx context.Context, req *pb.ListReactionsRequest) (*pb.ListReactionsResponse, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	reactions, err := s.chatService.ListReactions(ctx, messageID)
	if err != nil {
		return nil, handleError(err)
	}

	protoReactions := make([]*pb.Reaction, len(reactions))
	for i, r := range reactions {
		protoReactions[i] = &pb.Reaction{
			Id:        r.ID.String(),
			MessageId: r.MessageID.String(),
			UserId:    r.UserID.String(),
			Reaction:  r.Reaction,
			CreatedAt: timestamppb.New(r.CreatedAt),
		}
	}

	return &pb.ListReactionsResponse{
		Reactions: protoReactions,
	}, nil
}

// Read status operations

func (s *ChatServer) MarkAsRead(ctx context.Context, req *pb.MarkAsReadRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.MarkAsRead(ctx, chatID, messageID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) GetReadStatus(ctx context.Context, req *pb.GetReadStatusRequest) (*pb.ReadStatusResponse, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	readerIDs, count, err := s.chatService.GetReadStatus(ctx, messageID)
	if err != nil {
		return nil, handleError(err)
	}

	readers := make([]string, len(readerIDs))
	for i, id := range readerIDs {
		readers[i] = id.String()
	}

	return &pb.ReadStatusResponse{
		ReaderIds: readers,
		Count:     int32(count),
	}, nil
}

// Favorites operations

func (s *ChatServer) AddToFavorites(ctx context.Context, req *pb.AddToFavoritesRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.AddToFavorites(ctx, chatID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) RemoveFromFavorites(ctx context.Context, req *pb.RemoveFromFavoritesRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.RemoveFromFavorites(ctx, chatID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// Archive operations

func (s *ChatServer) ArchiveChat(ctx context.Context, req *pb.ArchiveChatRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.ArchiveChat(ctx, chatID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) UnarchiveChat(ctx context.Context, req *pb.UnarchiveChatRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.UnarchiveChat(ctx, chatID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *ChatServer) ListArchivedChats(ctx context.Context, req *pb.ListArchivedChatsRequest) (*pb.ListChatsResponse, error) {
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	count := int(req.Count)
	if count < 1 {
		count = 20
	}

	chats, total, err := s.chatService.ListArchivedChats(ctx, userID, page, count)
	if err != nil {
		return nil, handleError(err)
	}

	protoChats := make([]*pb.Chat, len(chats))
	for i, c := range chats {
		protoChats[i] = chatToProto(&c)
	}

	totalPages := int32(total) / int32(count)
	if int32(total)%int32(count) > 0 {
		totalPages++
	}

	return &pb.ListChatsResponse{
		Chats: protoChats,
		Pagination: &pb.Pagination{
			Page:       int32(page),
			Count:      int32(count),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}, nil
}

// Typing indicator

func (s *ChatServer) SendTyping(ctx context.Context, req *pb.SendTypingRequest) (*emptypb.Empty, error) {
	chatID, err := parseUUID(req.ChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid chat_id")
	}
	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.chatService.SendTyping(ctx, chatID, userID, req.IsTyping); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// Forward message

func (s *ChatServer) ForwardMessage(ctx context.Context, req *pb.ForwardMessageRequest) (*pb.Message, error) {
	messageID, err := parseUUID(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}
	targetChatID, err := parseUUID(req.TargetChatId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_chat_id")
	}
	senderID, err := parseUUID(req.SenderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid sender_id")
	}

	message, err := s.chatService.ForwardMessage(ctx, messageID, targetChatID, senderID)
	if err != nil {
		return nil, handleError(err)
	}

	return messageToProto(message), nil
}

// Poll operations - not implemented yet, using UnimplementedChatServiceServer
