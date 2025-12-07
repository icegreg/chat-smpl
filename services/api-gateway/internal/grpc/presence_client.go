package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	pb "github.com/icegreg/chat-smpl/proto/presence"
)

type PresenceClient struct {
	conn   *grpc.ClientConn
	client pb.PresenceServiceClient
}

func NewPresenceClient(addr string) (*PresenceClient, error) {
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

	return &PresenceClient{
		conn:   conn,
		client: pb.NewPresenceServiceClient(conn),
	}, nil
}

func (c *PresenceClient) Close() error {
	return c.conn.Close()
}

// SetStatus sets user's status
func (c *PresenceClient) SetStatus(ctx context.Context, userID string, status pb.UserStatus) (*pb.PresenceInfo, error) {
	resp, err := c.client.SetStatus(ctx, &pb.SetStatusRequest{
		UserId: userID,
		Status: status,
	})
	if err != nil {
		return nil, err
	}
	return resp.Presence, nil
}

// GetPresence gets presence info for a single user
func (c *PresenceClient) GetPresence(ctx context.Context, userID string) (*pb.PresenceInfo, error) {
	resp, err := c.client.GetPresence(ctx, &pb.GetPresenceRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Presence, nil
}

// GetPresencesBatch gets presence info for multiple users
func (c *PresenceClient) GetPresencesBatch(ctx context.Context, userIDs []string) ([]*pb.PresenceInfo, error) {
	resp, err := c.client.GetPresencesBatch(ctx, &pb.GetPresencesBatchRequest{
		UserIds: userIDs,
	})
	if err != nil {
		return nil, err
	}
	return resp.Presences, nil
}

// UserConnected is called when user establishes websocket connection
func (c *PresenceClient) UserConnected(ctx context.Context, userID, connectionID string) (*pb.PresenceInfo, error) {
	resp, err := c.client.UserConnected(ctx, &pb.UserConnectedRequest{
		UserId:       userID,
		ConnectionId: connectionID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Presence, nil
}

// UserDisconnected is called when user's websocket connection closes
func (c *PresenceClient) UserDisconnected(ctx context.Context, userID, connectionID string) (*pb.PresenceInfo, error) {
	resp, err := c.client.UserDisconnected(ctx, &pb.UserDisconnectedRequest{
		UserId:       userID,
		ConnectionId: connectionID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Presence, nil
}
