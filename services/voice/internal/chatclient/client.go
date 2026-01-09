package chatclient

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/icegreg/chat-smpl/proto/chat"
)

// ChatClient is a gRPC client for chat service
type ChatClient struct {
	conn   *grpc.ClientConn
	client pb.ChatServiceClient
}

// NewChatClient creates a new chat gRPC client
func NewChatClient(addr string) (*ChatClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	//nolint:staticcheck // grpc.Dial is deprecated but needed for grpc < 1.64
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}

	return &ChatClient{
		conn:   conn,
		client: pb.NewChatServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *ChatClient) Close() error {
	return c.conn.Close()
}

// SendSystemMessage sends a system message to a chat (appears in Activity thread)
func (c *ChatClient) SendSystemMessage(ctx context.Context, chatID, content string) (*pb.Message, error) {
	return c.client.SendSystemMessage(ctx, &pb.SendSystemMessageRequest{
		ChatId:  chatID,
		Content: content,
	})
}

// SendEventMessage sends a system message directly to the main chat (visible in conference chat panel)
func (c *ChatClient) SendEventMessage(ctx context.Context, chatID, content string) (*pb.Message, error) {
	return c.client.SendSystemMessage(ctx, &pb.SendSystemMessageRequest{
		ChatId:     chatID,
		Content:    content,
		ToMainChat: true,
	})
}

// CreateThread creates a new thread in a chat
func (c *ChatClient) CreateThread(ctx context.Context, chatID string, threadType pb.ThreadType, title string, createdBy string) (*pb.Thread, error) {
	return c.client.CreateThread(ctx, &pb.CreateThreadRequest{
		ChatId:     chatID,
		ThreadType: threadType,
		Title:      title,
		CreatedBy:  createdBy,
	})
}
