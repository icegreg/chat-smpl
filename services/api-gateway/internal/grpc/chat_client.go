package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/icegreg/chat-smpl/proto/chat"
)

type ChatClient struct {
	conn   *grpc.ClientConn
	client pb.ChatServiceClient
}

func NewChatClient(addr string) (*ChatClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &ChatClient{
		conn:   conn,
		client: pb.NewChatServiceClient(conn),
	}, nil
}

func (c *ChatClient) Close() error {
	return c.conn.Close()
}

// Chat operations

func (c *ChatClient) CreateChat(ctx context.Context, userID string, chatType pb.ChatType, name string, participantIDs []string) (*pb.Chat, error) {
	return c.client.CreateChat(ctx, &pb.CreateChatRequest{
		Name:           name,
		ChatType:       chatType,
		CreatedBy:      userID,
		ParticipantIds: participantIDs,
	})
}

func (c *ChatClient) GetChat(ctx context.Context, chatID, userID string) (*pb.Chat, error) {
	return c.client.GetChat(ctx, &pb.GetChatRequest{
		ChatId: chatID,
		UserId: userID,
	})
}

func (c *ChatClient) ListChats(ctx context.Context, userID string, page, count int32) (*pb.ListChatsResponse, error) {
	return c.client.ListChats(ctx, &pb.ListChatsRequest{
		UserId: userID,
		Page:   page,
		Count:  count,
	})
}

func (c *ChatClient) UpdateChat(ctx context.Context, chatID, userID, name string) (*pb.Chat, error) {
	return c.client.UpdateChat(ctx, &pb.UpdateChatRequest{
		ChatId: chatID,
		Name:   name,
		UserId: userID,
	})
}

func (c *ChatClient) DeleteChat(ctx context.Context, chatID, userID string) error {
	_, err := c.client.DeleteChat(ctx, &pb.DeleteChatRequest{
		ChatId: chatID,
		UserId: userID,
	})
	return err
}

func (c *ChatClient) SearchChats(ctx context.Context, userID, query string, page, count int32) (*pb.ListChatsResponse, error) {
	return c.client.SearchChats(ctx, &pb.SearchChatsRequest{
		UserId: userID,
		Query:  query,
		Page:   page,
		Count:  count,
	})
}

// Participant operations

func (c *ChatClient) AddParticipant(ctx context.Context, chatID, userID, addedBy string, role pb.ParticipantRole) (*pb.ChatParticipant, error) {
	return c.client.AddParticipant(ctx, &pb.AddParticipantRequest{
		ChatId:  chatID,
		UserId:  userID,
		AddedBy: addedBy,
		Role:    role,
	})
}

func (c *ChatClient) RemoveParticipant(ctx context.Context, chatID, userID, removedBy string) error {
	_, err := c.client.RemoveParticipant(ctx, &pb.RemoveParticipantRequest{
		ChatId:    chatID,
		UserId:    userID,
		RemovedBy: removedBy,
	})
	return err
}

func (c *ChatClient) UpdateParticipantRole(ctx context.Context, chatID, userID, updatedBy string, role pb.ParticipantRole) (*pb.ChatParticipant, error) {
	return c.client.UpdateParticipantRole(ctx, &pb.UpdateParticipantRoleRequest{
		ChatId:    chatID,
		UserId:    userID,
		UpdatedBy: updatedBy,
		Role:      role,
	})
}

func (c *ChatClient) ListParticipants(ctx context.Context, chatID string, page, count int32) (*pb.ListParticipantsResponse, error) {
	return c.client.ListParticipants(ctx, &pb.ListParticipantsRequest{
		ChatId: chatID,
		Page:   page,
		Count:  count,
	})
}

// Message operations

func (c *ChatClient) SendMessage(ctx context.Context, chatID, senderID, content string, parentID string, fileLinkIDs []string) (*pb.Message, error) {
	return c.client.SendMessage(ctx, &pb.SendMessageRequest{
		ChatId:      chatID,
		SenderId:    senderID,
		Content:     content,
		ParentId:    parentID,
		FileLinkIds: fileLinkIDs,
	})
}

func (c *ChatClient) GetMessage(ctx context.Context, messageID, userID string) (*pb.Message, error) {
	return c.client.GetMessage(ctx, &pb.GetMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
}

func (c *ChatClient) ListMessages(ctx context.Context, chatID, userID string, page, count int32) (*pb.ListMessagesResponse, error) {
	return c.client.ListMessages(ctx, &pb.ListMessagesRequest{
		ChatId: chatID,
		UserId: userID,
		Page:   page,
		Count:  count,
	})
}

func (c *ChatClient) UpdateMessage(ctx context.Context, messageID, userID, content string) (*pb.Message, error) {
	return c.client.UpdateMessage(ctx, &pb.UpdateMessageRequest{
		MessageId: messageID,
		UserId:    userID,
		Content:   content,
	})
}

func (c *ChatClient) DeleteMessage(ctx context.Context, messageID, userID string) error {
	_, err := c.client.DeleteMessage(ctx, &pb.DeleteMessageRequest{
		MessageId: messageID,
		UserId:    userID,
	})
	return err
}

func (c *ChatClient) GetThreadMessages(ctx context.Context, parentID, userID string, page, count int32) (*pb.ListMessagesResponse, error) {
	return c.client.GetThreadMessages(ctx, &pb.GetThreadMessagesRequest{
		ParentId: parentID,
		UserId:   userID,
		Page:     page,
		Count:    count,
	})
}

// Reaction operations

func (c *ChatClient) AddReaction(ctx context.Context, messageID, userID, reaction string) error {
	_, err := c.client.AddReaction(ctx, &pb.AddReactionRequest{
		MessageId: messageID,
		UserId:    userID,
		Reaction:  reaction,
	})
	return err
}

func (c *ChatClient) RemoveReaction(ctx context.Context, messageID, userID, reaction string) error {
	_, err := c.client.RemoveReaction(ctx, &pb.RemoveReactionRequest{
		MessageId: messageID,
		UserId:    userID,
		Reaction:  reaction,
	})
	return err
}

func (c *ChatClient) ListReactions(ctx context.Context, messageID string) (*pb.ListReactionsResponse, error) {
	return c.client.ListReactions(ctx, &pb.ListReactionsRequest{
		MessageId: messageID,
	})
}

// Read status operations

func (c *ChatClient) MarkAsRead(ctx context.Context, chatID, messageID, userID string) error {
	_, err := c.client.MarkAsRead(ctx, &pb.MarkAsReadRequest{
		ChatId:    chatID,
		MessageId: messageID,
		UserId:    userID,
	})
	return err
}

func (c *ChatClient) GetReadStatus(ctx context.Context, messageID string) (*pb.ReadStatusResponse, error) {
	return c.client.GetReadStatus(ctx, &pb.GetReadStatusRequest{
		MessageId: messageID,
	})
}

// Favorites operations

func (c *ChatClient) AddToFavorites(ctx context.Context, chatID, userID string) error {
	_, err := c.client.AddToFavorites(ctx, &pb.AddToFavoritesRequest{
		ChatId: chatID,
		UserId: userID,
	})
	return err
}

func (c *ChatClient) RemoveFromFavorites(ctx context.Context, chatID, userID string) error {
	_, err := c.client.RemoveFromFavorites(ctx, &pb.RemoveFromFavoritesRequest{
		ChatId: chatID,
		UserId: userID,
	})
	return err
}

// Archive operations

func (c *ChatClient) ArchiveChat(ctx context.Context, chatID, userID string) error {
	_, err := c.client.ArchiveChat(ctx, &pb.ArchiveChatRequest{
		ChatId: chatID,
		UserId: userID,
	})
	return err
}

func (c *ChatClient) UnarchiveChat(ctx context.Context, chatID, userID string) error {
	_, err := c.client.UnarchiveChat(ctx, &pb.UnarchiveChatRequest{
		ChatId: chatID,
		UserId: userID,
	})
	return err
}

func (c *ChatClient) ListArchivedChats(ctx context.Context, userID string, page, count int32) (*pb.ListChatsResponse, error) {
	return c.client.ListArchivedChats(ctx, &pb.ListArchivedChatsRequest{
		UserId: userID,
		Page:   page,
		Count:  count,
	})
}

// Poll operations

func (c *ChatClient) CreatePoll(ctx context.Context, chatID, createdBy, question string, options []string, isMultipleChoice, isAnonymous bool) (*pb.Poll, error) {
	return c.client.CreatePoll(ctx, &pb.CreatePollRequest{
		ChatId:           chatID,
		CreatedBy:        createdBy,
		Question:         question,
		Options:          options,
		IsMultipleChoice: isMultipleChoice,
		IsAnonymous:      isAnonymous,
	})
}

func (c *ChatClient) VotePoll(ctx context.Context, pollID, userID string, optionIDs []string) error {
	_, err := c.client.VotePoll(ctx, &pb.VotePollRequest{
		PollId:    pollID,
		UserId:    userID,
		OptionIds: optionIDs,
	})
	return err
}

func (c *ChatClient) FinishPoll(ctx context.Context, pollID, userID string) (*pb.Poll, error) {
	return c.client.FinishPoll(ctx, &pb.FinishPollRequest{
		PollId: pollID,
		UserId: userID,
	})
}

func (c *ChatClient) DeletePoll(ctx context.Context, pollID, userID string) error {
	_, err := c.client.DeletePoll(ctx, &pb.DeletePollRequest{
		PollId: pollID,
		UserId: userID,
	})
	return err
}

// Typing indicator

func (c *ChatClient) SendTyping(ctx context.Context, chatID, userID string, isTyping bool) error {
	_, err := c.client.SendTyping(ctx, &pb.SendTypingRequest{
		ChatId:   chatID,
		UserId:   userID,
		IsTyping: isTyping,
	})
	return err
}
