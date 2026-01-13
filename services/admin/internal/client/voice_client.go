package client

import (
	"context"
	"fmt"

	pb "github.com/icegreg/chat-smpl/proto/voice"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// VoiceClient is a client for voice-service gRPC
type VoiceClient struct {
	conn   *grpc.ClientConn
	client pb.VoiceServiceClient
	logger *zap.Logger
}

// NewVoiceClient creates a new voice service client
func NewVoiceClient(addr string, logger *zap.Logger) (*VoiceClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to voice service: %w", err)
	}

	return &VoiceClient{
		conn:   conn,
		client: pb.NewVoiceServiceClient(conn),
		logger: logger,
	}, nil
}

// Close closes the gRPC connection
func (c *VoiceClient) Close() error {
	return c.conn.Close()
}

// GetConference gets a conference by ID
func (c *VoiceClient) GetConference(ctx context.Context, conferenceID string, userID string) (*pb.Conference, error) {
	req := &pb.GetConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
	}

	conf, err := c.client.GetConference(ctx, req)
	if err != nil {
		c.logger.Error("failed to get conference", zap.Error(err), zap.String("conference_id", conferenceID))
		return nil, err
	}

	return conf, nil
}

// ListActiveConferences lists all active conferences
func (c *VoiceClient) ListActiveConferences(ctx context.Context, userID string) ([]*pb.Conference, error) {
	req := &pb.ListConferencesRequest{
		UserId:     userID,
		ActiveOnly: true,
	}

	resp, err := c.client.ListConferences(ctx, req)
	if err != nil {
		c.logger.Error("failed to list active conferences", zap.Error(err))
		return nil, err
	}

	return resp.Conferences, nil
}

// GetParticipants gets participants of a conference
func (c *VoiceClient) GetParticipants(ctx context.Context, conferenceID string, userID string) ([]*pb.Participant, error) {
	req := &pb.GetConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
	}

	conf, err := c.client.GetConference(ctx, req)
	if err != nil {
		c.logger.Error("failed to get conference participants", zap.Error(err), zap.String("conference_id", conferenceID))
		return nil, err
	}

	return conf.Participants, nil
}

// EndConference ends a conference
func (c *VoiceClient) EndConference(ctx context.Context, conferenceID string, userID string) error {
	req := &pb.EndConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
	}

	_, err := c.client.EndConference(ctx, req)
	if err != nil {
		c.logger.Error("failed to end conference", zap.Error(err), zap.String("conference_id", conferenceID))
		return err
	}

	return nil
}
