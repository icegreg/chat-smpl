package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/icegreg/chat-smpl/proto/voice"
)

type VoiceClient struct {
	conn   *grpc.ClientConn
	client pb.VoiceServiceClient
}

func NewVoiceClient(addr string) (*VoiceClient, error) {
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

	return &VoiceClient{
		conn:   conn,
		client: pb.NewVoiceServiceClient(conn),
	}, nil
}

func (c *VoiceClient) Close() error {
	return c.conn.Close()
}

// CreateConference creates a new conference
func (c *VoiceClient) CreateConference(ctx context.Context, req *pb.CreateConferenceRequest) (*pb.Conference, error) {
	return c.client.CreateConference(ctx, req)
}

// GetConference retrieves a conference by ID
func (c *VoiceClient) GetConference(ctx context.Context, conferenceID string) (*pb.Conference, error) {
	return c.client.GetConference(ctx, &pb.GetConferenceRequest{
		ConferenceId: conferenceID,
	})
}

// ListConferences lists conferences for a user
func (c *VoiceClient) ListConferences(ctx context.Context, userID string, activeOnly bool, limit, offset int32) (*pb.ListConferencesResponse, error) {
	return c.client.ListConferences(ctx, &pb.ListConferencesRequest{
		UserId:     userID,
		ActiveOnly: activeOnly,
		Limit:      limit,
		Offset:     offset,
	})
}

// JoinConference joins a user to a conference
func (c *VoiceClient) JoinConference(ctx context.Context, conferenceID, userID string, muted bool) (*pb.Participant, error) {
	return c.client.JoinConference(ctx, &pb.JoinConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
		Muted:        muted,
	})
}

// LeaveConference removes a user from a conference
func (c *VoiceClient) LeaveConference(ctx context.Context, conferenceID, userID string) error {
	_, err := c.client.LeaveConference(ctx, &pb.LeaveConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
	})
	return err
}

// GetParticipants returns all participants in a conference
func (c *VoiceClient) GetParticipants(ctx context.Context, conferenceID string) (*pb.GetParticipantsResponse, error) {
	return c.client.GetParticipants(ctx, &pb.GetParticipantsRequest{
		ConferenceId: conferenceID,
	})
}

// MuteParticipant mutes or unmutes a participant
// actorUserID is the user performing the action, targetUserID is the user to mute/unmute
func (c *VoiceClient) MuteParticipant(ctx context.Context, conferenceID, actorUserID, targetUserID string, mute bool) (*pb.Participant, error) {
	return c.client.MuteParticipant(ctx, &pb.MuteParticipantRequest{
		ConferenceId: conferenceID,
		UserId:       actorUserID,
		TargetUserId: targetUserID,
		Mute:         mute,
	})
}

// KickParticipant removes a participant from conference
// actorUserID is the user performing the action, targetUserID is the user to kick
func (c *VoiceClient) KickParticipant(ctx context.Context, conferenceID, actorUserID, targetUserID string) error {
	_, err := c.client.KickParticipant(ctx, &pb.KickParticipantRequest{
		ConferenceId: conferenceID,
		UserId:       actorUserID,
		TargetUserId: targetUserID,
	})
	return err
}

// EndConference ends a conference
func (c *VoiceClient) EndConference(ctx context.Context, conferenceID, userID string) error {
	_, err := c.client.EndConference(ctx, &pb.EndConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID,
	})
	return err
}

// InitiateCall starts a 1-on-1 call
func (c *VoiceClient) InitiateCall(ctx context.Context, callerID, calleeID string, chatID string) (*pb.Call, error) {
	return c.client.InitiateCall(ctx, &pb.InitiateCallRequest{
		CallerId: callerID,
		CalleeId: calleeID,
		ChatId:   chatID,
	})
}

// AnswerCall answers an incoming call
func (c *VoiceClient) AnswerCall(ctx context.Context, callID, userID string) (*pb.Call, error) {
	return c.client.AnswerCall(ctx, &pb.AnswerCallRequest{
		CallId: callID,
		UserId: userID,
	})
}

// HangupCall ends a call
func (c *VoiceClient) HangupCall(ctx context.Context, callID, userID string) error {
	_, err := c.client.HangupCall(ctx, &pb.HangupCallRequest{
		CallId: callID,
		UserId: userID,
	})
	return err
}

// GetCallHistory returns call history for a user
func (c *VoiceClient) GetCallHistory(ctx context.Context, userID string, limit, offset int32) (*pb.GetCallHistoryResponse, error) {
	return c.client.GetCallHistory(ctx, &pb.GetCallHistoryRequest{
		UserId: userID,
		Limit:  limit,
		Offset: offset,
	})
}

// GetVertoCredentials generates Verto credentials for a user
func (c *VoiceClient) GetVertoCredentials(ctx context.Context, userID string) (*pb.VertoCredentials, error) {
	return c.client.GetVertoCredentials(ctx, &pb.GetVertoCredentialsRequest{
		UserId: userID,
	})
}

// StartChatCall starts a call from a chat room
func (c *VoiceClient) StartChatCall(ctx context.Context, chatID, userID string) (*pb.StartChatCallResponse, error) {
	return c.client.StartChatCall(ctx, &pb.StartChatCallRequest{
		ChatId: chatID,
		UserId: userID,
	})
}

// ======== Scheduled Events ========

// ScheduleConference creates a scheduled or recurring conference
func (c *VoiceClient) ScheduleConference(ctx context.Context, req *pb.ScheduleConferenceRequest) (*pb.Conference, error) {
	return c.client.ScheduleConference(ctx, req)
}

// CreateAdHocFromChat creates an ad-hoc call from a chat
func (c *VoiceClient) CreateAdHocFromChat(ctx context.Context, req *pb.CreateAdHocFromChatRequest) (*pb.Conference, error) {
	return c.client.CreateAdHocFromChat(ctx, req)
}

// UpdateRSVP updates a participant's RSVP status
func (c *VoiceClient) UpdateRSVP(ctx context.Context, req *pb.UpdateRSVPRequest) (*pb.Participant, error) {
	return c.client.UpdateRSVP(ctx, req)
}

// UpdateParticipantRole updates a participant's role
func (c *VoiceClient) UpdateParticipantRole(ctx context.Context, req *pb.UpdateParticipantRoleRequest) (*pb.Participant, error) {
	return c.client.UpdateParticipantRole(ctx, req)
}

// AddParticipants adds participants to a conference
func (c *VoiceClient) AddParticipants(ctx context.Context, req *pb.AddParticipantsRequest) (*emptypb.Empty, error) {
	return c.client.AddParticipants(ctx, req)
}

// RemoveParticipant removes a participant from a conference
func (c *VoiceClient) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*emptypb.Empty, error) {
	return c.client.RemoveParticipant(ctx, req)
}

// ListScheduledConferences lists scheduled conferences for a user
func (c *VoiceClient) ListScheduledConferences(ctx context.Context, req *pb.ListScheduledConferencesRequest) (*pb.ListScheduledConferencesResponse, error) {
	return c.client.ListScheduledConferences(ctx, req)
}

// GetChatConferences gets conferences for a specific chat
func (c *VoiceClient) GetChatConferences(ctx context.Context, req *pb.GetChatConferencesRequest) (*pb.GetChatConferencesResponse, error) {
	return c.client.GetChatConferences(ctx, req)
}

// CancelConference cancels a scheduled conference
func (c *VoiceClient) CancelConference(ctx context.Context, req *pb.CancelConferenceRequest) (*emptypb.Empty, error) {
	return c.client.CancelConference(ctx, req)
}
