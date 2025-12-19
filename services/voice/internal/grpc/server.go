package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/icegreg/chat-smpl/services/voice/internal/model"
	"github.com/icegreg/chat-smpl/services/voice/internal/repository"
	"github.com/icegreg/chat-smpl/services/voice/internal/service"
	pb "github.com/icegreg/chat-smpl/proto/voice"
)

// Server implements the VoiceService gRPC server
type Server struct {
	pb.UnimplementedVoiceServiceServer
	voiceService service.VoiceService
	logger       *zap.Logger
}

// NewServer creates a new gRPC server
func NewServer(voiceService service.VoiceService, logger *zap.Logger) *Server {
	return &Server{
		voiceService: voiceService,
		logger:       logger,
	}
}

// CreateConference creates a new conference
func (s *Server) CreateConference(ctx context.Context, req *pb.CreateConferenceRequest) (*pb.Conference, error) {
	createdBy, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", err)
	}

	var chatID *uuid.UUID
	if req.GetChatId() != "" {
		parsed, err := uuid.Parse(req.GetChatId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
		}
		chatID = &parsed
	}

	conf, err := s.voiceService.CreateConference(ctx, &model.CreateConferenceRequest{
		Name:            req.GetName(),
		ChatID:          chatID,
		CreatedBy:       createdBy,
		MaxMembers:      int(req.GetMaxMembers()),
		IsPrivate:       req.GetIsPrivate(),
		EnableRecording: req.GetEnableRecording(),
	})
	if err != nil {
		s.logger.Error("failed to create conference", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create conference: %v", err)
	}

	return conferenceToProto(conf), nil
}

// GetConference retrieves a conference by ID
func (s *Server) GetConference(ctx context.Context, req *pb.GetConferenceRequest) (*pb.Conference, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	conf, err := s.voiceService.GetConference(ctx, confID)
	if err != nil {
		if errors.Is(err, repository.ErrConferenceNotFound) {
			return nil, status.Error(codes.NotFound, "conference not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get conference: %v", err)
	}

	return conferenceToProto(conf), nil
}

// ListConferences lists conferences for a user
func (s *Server) ListConferences(ctx context.Context, req *pb.ListConferencesRequest) (*pb.ListConferencesResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 20
	}

	conferences, total, err := s.voiceService.ListConferences(ctx, userID, req.GetActiveOnly(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conferences: %v", err)
	}

	protoConfs := make([]*pb.Conference, len(conferences))
	for i, conf := range conferences {
		protoConfs[i] = conferenceToProto(conf)
	}

	return &pb.ListConferencesResponse{
		Conferences: protoConfs,
		Total:       int32(total),
	}, nil
}

// JoinConference joins a user to a conference
func (s *Server) JoinConference(ctx context.Context, req *pb.JoinConferenceRequest) (*pb.Participant, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	participant, err := s.voiceService.JoinConference(ctx, confID, userID, model.JoinOptions{
		Muted: req.GetMuted(),
	})
	if err != nil {
		if errors.Is(err, repository.ErrConferenceNotFound) {
			return nil, status.Error(codes.NotFound, "conference not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to join conference: %v", err)
	}

	return participantToProto(participant), nil
}

// LeaveConference removes a user from a conference
func (s *Server) LeaveConference(ctx context.Context, req *pb.LeaveConferenceRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	if err := s.voiceService.LeaveConference(ctx, confID, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to leave conference: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetParticipants returns all participants in a conference
func (s *Server) GetParticipants(ctx context.Context, req *pb.GetParticipantsRequest) (*pb.GetParticipantsResponse, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	participants, err := s.voiceService.GetParticipants(ctx, confID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get participants: %v", err)
	}

	protoParticipants := make([]*pb.Participant, len(participants))
	for i, p := range participants {
		protoParticipants[i] = participantToProto(p)
	}

	return &pb.GetParticipantsResponse{
		Participants: protoParticipants,
	}, nil
}

// MuteParticipant mutes or unmutes a participant
func (s *Server) MuteParticipant(ctx context.Context, req *pb.MuteParticipantRequest) (*pb.Participant, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	// user_id is the actor performing the mute
	actorUserID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	// target_user_id is the user to mute/unmute
	targetUserID, err := uuid.Parse(req.GetTargetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_user_id: %v", err)
	}

	participant, err := s.voiceService.MuteParticipant(ctx, confID, targetUserID, actorUserID, req.GetMute())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mute participant: %v", err)
	}

	return participantToProto(participant), nil
}

// KickParticipant removes a participant from conference
func (s *Server) KickParticipant(ctx context.Context, req *pb.KickParticipantRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	// user_id is the actor performing the kick
	actorUserID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	// target_user_id is the user to kick
	targetUserID, err := uuid.Parse(req.GetTargetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_user_id: %v", err)
	}

	if err := s.voiceService.KickParticipant(ctx, confID, targetUserID, actorUserID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to kick participant: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// EndConference ends a conference
func (s *Server) EndConference(ctx context.Context, req *pb.EndConferenceRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	if err := s.voiceService.EndConference(ctx, confID, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to end conference: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// InitiateCall starts a 1-on-1 call
func (s *Server) InitiateCall(ctx context.Context, req *pb.InitiateCallRequest) (*pb.Call, error) {
	callerID, err := uuid.Parse(req.GetCallerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid caller_id: %v", err)
	}

	calleeID, err := uuid.Parse(req.GetCalleeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid callee_id: %v", err)
	}

	var chatID *uuid.UUID
	if req.GetChatId() != "" {
		parsed, err := uuid.Parse(req.GetChatId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
		}
		chatID = &parsed
	}

	call, err := s.voiceService.InitiateCall(ctx, callerID, calleeID, chatID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to initiate call: %v", err)
	}

	return callToProto(call), nil
}

// AnswerCall answers an incoming call
func (s *Server) AnswerCall(ctx context.Context, req *pb.AnswerCallRequest) (*pb.Call, error) {
	callID, err := uuid.Parse(req.GetCallId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid call_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	call, err := s.voiceService.AnswerCall(ctx, callID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrCallNotFound) {
			return nil, status.Error(codes.NotFound, "call not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to answer call: %v", err)
	}

	return callToProto(call), nil
}

// HangupCall ends a call
func (s *Server) HangupCall(ctx context.Context, req *pb.HangupCallRequest) (*emptypb.Empty, error) {
	callID, err := uuid.Parse(req.GetCallId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid call_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	if err := s.voiceService.HangupCall(ctx, callID, userID); err != nil {
		if errors.Is(err, repository.ErrCallNotFound) {
			return nil, status.Error(codes.NotFound, "call not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to hangup call: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// GetCallHistory returns call history for a user
func (s *Server) GetCallHistory(ctx context.Context, req *pb.GetCallHistoryRequest) (*pb.GetCallHistoryResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 20
	}

	calls, total, err := s.voiceService.GetCallHistory(ctx, userID, limit, int(req.GetOffset()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get call history: %v", err)
	}

	protoCalls := make([]*pb.Call, len(calls))
	for i, call := range calls {
		protoCalls[i] = callToProto(call)
	}

	return &pb.GetCallHistoryResponse{
		Calls: protoCalls,
		Total: int32(total),
	}, nil
}

// GetVertoCredentials generates Verto credentials for a user
func (s *Server) GetVertoCredentials(ctx context.Context, req *pb.GetVertoCredentialsRequest) (*pb.VertoCredentials, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	creds, err := s.voiceService.GetVertoCredentials(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get verto credentials: %v", err)
	}

	return vertoCredentialsToProto(creds), nil
}

// StartChatCall starts a call from a chat room
func (s *Server) StartChatCall(ctx context.Context, req *pb.StartChatCallRequest) (*pb.StartChatCallResponse, error) {
	chatID, err := uuid.Parse(req.GetChatId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	conf, creds, err := s.voiceService.StartChatCall(ctx, chatID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start chat call: %v", err)
	}

	return &pb.StartChatCallResponse{
		Conference:  conferenceToProto(conf),
		Credentials: vertoCredentialsToProto(creds),
	}, nil
}

// ======== Scheduled Events ========

// ScheduleConference creates a scheduled or recurring conference
func (s *Server) ScheduleConference(ctx context.Context, req *pb.ScheduleConferenceRequest) (*pb.Conference, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	var chatID *uuid.UUID
	if req.GetChatId() != "" {
		parsed, err := uuid.Parse(req.GetChatId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
		}
		chatID = &parsed
	}

	participantIDs := make([]uuid.UUID, 0, len(req.GetParticipantUserIds()))
	for _, id := range req.GetParticipantUserIds() {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.logger.Warn("skipping invalid participant_id", zap.String("id", id), zap.Error(err))
			continue
		}
		participantIDs = append(participantIDs, parsed)
	}

	var recurrence *model.RecurrenceRule
	if req.GetRecurrence() != nil {
		recurrence = recurrenceRuleFromProto(req.GetRecurrence())
	}

	conf, err := s.voiceService.ScheduleConference(ctx, &model.ScheduleConferenceRequest{
		Name:            req.GetName(),
		ChatID:          chatID,
		UserID:          userID,
		ScheduledAt:     req.GetScheduledAt().AsTime(),
		Recurrence:      recurrence,
		ParticipantIDs:  participantIDs,
		MaxMembers:      int(req.GetMaxMembers()),
		EnableRecording: req.GetEnableRecording(),
	})
	if err != nil {
		s.logger.Error("failed to schedule conference", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to schedule conference: %v", err)
	}

	return conferenceToProto(conf), nil
}

// CreateAdHocFromChat creates an ad-hoc call from a chat
func (s *Server) CreateAdHocFromChat(ctx context.Context, req *pb.CreateAdHocFromChatRequest) (*pb.Conference, error) {
	chatID, err := uuid.Parse(req.GetChatId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	participantIDs := make([]uuid.UUID, 0, len(req.GetParticipantUserIds()))
	for _, id := range req.GetParticipantUserIds() {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.logger.Warn("skipping invalid participant_id", zap.String("id", id), zap.Error(err))
			continue
		}
		participantIDs = append(participantIDs, parsed)
	}

	conf, err := s.voiceService.CreateAdHocFromChat(ctx, &model.CreateAdHocFromChatRequest{
		ChatID:         chatID,
		UserID:         userID,
		ParticipantIDs: participantIDs,
	})
	if err != nil {
		s.logger.Error("failed to create ad-hoc conference", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create ad-hoc conference: %v", err)
	}

	return conferenceToProto(conf), nil
}

// UpdateRSVP updates a participant's RSVP status
func (s *Server) UpdateRSVP(ctx context.Context, req *pb.UpdateRSVPRequest) (*pb.Participant, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	participant, err := s.voiceService.UpdateRSVP(ctx, &model.UpdateRSVPRequest{
		ConferenceID: confID,
		UserID:       userID,
		RSVPStatus:   rsvpStatusFromProto(req.GetRsvpStatus()),
	})
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, status.Error(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update RSVP: %v", err)
	}

	return participantToProto(participant), nil
}

// UpdateParticipantRole updates a participant's role
func (s *Server) UpdateParticipantRole(ctx context.Context, req *pb.UpdateParticipantRoleRequest) (*pb.Participant, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	actorUserID, err := uuid.Parse(req.GetActorUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid actor_user_id: %v", err)
	}

	targetUserID, err := uuid.Parse(req.GetTargetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_user_id: %v", err)
	}

	participant, err := s.voiceService.UpdateParticipantRole(ctx, &model.UpdateParticipantRoleRequest{
		ConferenceID: confID,
		ActorUserID:  actorUserID,
		TargetUserID: targetUserID,
		NewRole:      conferenceRoleFromProto(req.GetNewRole()),
	})
	if err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, status.Error(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update role: %v", err)
	}

	return participantToProto(participant), nil
}

// AddParticipants adds participants to a conference
func (s *Server) AddParticipants(ctx context.Context, req *pb.AddParticipantsRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	actorUserID, err := uuid.Parse(req.GetActorUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid actor_user_id: %v", err)
	}

	userIDs := make([]uuid.UUID, 0, len(req.GetUserIds()))
	for _, id := range req.GetUserIds() {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.logger.Warn("skipping invalid user_id", zap.String("id", id), zap.Error(err))
			continue
		}
		userIDs = append(userIDs, parsed)
	}

	if err := s.voiceService.AddParticipants(ctx, &model.AddParticipantsRequest{
		ConferenceID: confID,
		ActorUserID:  actorUserID,
		UserIDs:      userIDs,
		DefaultRole:  conferenceRoleFromProto(req.GetDefaultRole()),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add participants: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// RemoveParticipant removes a participant from a conference
func (s *Server) RemoveParticipant(ctx context.Context, req *pb.RemoveParticipantRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	actorUserID, err := uuid.Parse(req.GetActorUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid actor_user_id: %v", err)
	}

	targetUserID, err := uuid.Parse(req.GetTargetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_user_id: %v", err)
	}

	if err := s.voiceService.RemoveParticipant(ctx, &model.RemoveParticipantRequest{
		ConferenceID: confID,
		ActorUserID:  actorUserID,
		TargetUserID: targetUserID,
	}); err != nil {
		if errors.Is(err, repository.ErrParticipantNotFound) {
			return nil, status.Error(codes.NotFound, "participant not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to remove participant: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListScheduledConferences lists scheduled conferences for a user
func (s *Server) ListScheduledConferences(ctx context.Context, req *pb.ListScheduledConferencesRequest) (*pb.ListScheduledConferencesResponse, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	limit := int(req.GetLimit())
	if limit == 0 {
		limit = 20
	}

	conferences, total, err := s.voiceService.ListScheduledConferences(ctx, userID, req.GetUpcomingOnly(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list scheduled conferences: %v", err)
	}

	protoConfs := make([]*pb.Conference, len(conferences))
	for i, conf := range conferences {
		protoConfs[i] = conferenceToProto(conf)
	}

	return &pb.ListScheduledConferencesResponse{
		Conferences: protoConfs,
		Total:       int32(total),
	}, nil
}

// GetChatConferences gets conferences for a specific chat
func (s *Server) GetChatConferences(ctx context.Context, req *pb.GetChatConferencesRequest) (*pb.GetChatConferencesResponse, error) {
	chatID, err := uuid.Parse(req.GetChatId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid chat_id: %v", err)
	}

	conferences, err := s.voiceService.GetChatConferences(ctx, chatID, req.GetUpcomingOnly())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get chat conferences: %v", err)
	}

	protoConfs := make([]*pb.Conference, len(conferences))
	for i, conf := range conferences {
		protoConfs[i] = conferenceToProto(conf)
	}

	return &pb.GetChatConferencesResponse{
		Conferences: protoConfs,
	}, nil
}

// CancelConference cancels a scheduled conference
func (s *Server) CancelConference(ctx context.Context, req *pb.CancelConferenceRequest) (*emptypb.Empty, error) {
	confID, err := uuid.Parse(req.GetConferenceId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid conference_id: %v", err)
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	if err := s.voiceService.CancelConference(ctx, confID, userID, req.GetCancelSeries()); err != nil {
		if errors.Is(err, repository.ErrConferenceNotFound) {
			return nil, status.Error(codes.NotFound, "conference not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to cancel conference: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// Helper functions to convert models to protobuf

func conferenceToProto(conf *model.Conference) *pb.Conference {
	proto := &pb.Conference{
		Id:               conf.ID.String(),
		Name:             conf.Name,
		CreatedBy:        conf.CreatedBy.String(),
		Status:           conferenceStatusToProto(conf.Status),
		MaxMembers:       int32(conf.MaxMembers),
		ParticipantCount: int32(conf.ParticipantCount),
		IsPrivate:        conf.IsPrivate,
		CreatedAt:        timestamppb.New(conf.CreatedAt),
		// Scheduled events fields
		EventType:     eventTypeToProto(conf.EventType),
		AcceptedCount: int32(conf.AcceptedCount),
		DeclinedCount: int32(conf.DeclinedCount),
	}
	if conf.ChatID != nil {
		proto.ChatId = conf.ChatID.String()
	}
	if conf.RecordingPath != nil {
		proto.RecordingPath = *conf.RecordingPath
	}
	if conf.StartedAt != nil {
		proto.StartedAt = timestamppb.New(*conf.StartedAt)
	}
	if conf.EndedAt != nil {
		proto.EndedAt = timestamppb.New(*conf.EndedAt)
	}
	if conf.ScheduledAt != nil {
		proto.ScheduledAt = timestamppb.New(*conf.ScheduledAt)
	}
	if conf.SeriesID != nil {
		proto.SeriesId = conf.SeriesID.String()
	}
	if conf.Recurrence != nil {
		proto.Recurrence = recurrenceRuleToProto(conf.Recurrence)
	}
	// Convert participants
	if len(conf.Participants) > 0 {
		proto.Participants = make([]*pb.Participant, len(conf.Participants))
		for i, p := range conf.Participants {
			proto.Participants[i] = participantToProto(&p)
		}
	}
	return proto
}

func conferenceStatusToProto(status model.ConferenceStatus) pb.ConferenceStatus {
	switch status {
	case model.ConferenceStatusActive:
		return pb.ConferenceStatus_CONFERENCE_STATUS_ACTIVE
	case model.ConferenceStatusEnded:
		return pb.ConferenceStatus_CONFERENCE_STATUS_ENDED
	case model.ConferenceStatusScheduled:
		return pb.ConferenceStatus_CONFERENCE_STATUS_SCHEDULED
	case model.ConferenceStatusCancelled:
		return pb.ConferenceStatus_CONFERENCE_STATUS_CANCELLED
	default:
		return pb.ConferenceStatus_CONFERENCE_STATUS_UNSPECIFIED
	}
}

func participantToProto(p *model.Participant) *pb.Participant {
	proto := &pb.Participant{
		Id:           p.ID.String(),
		ConferenceId: p.ConferenceID.String(),
		UserId:       p.UserID.String(),
		Status:       participantStatusToProto(p.Status),
		IsMuted:      p.IsMuted,
		IsDeaf:       p.IsDeaf,
		IsSpeaking:   p.IsSpeaking,
		// Scheduled events fields
		Role:       conferenceRoleToProto(p.Role),
		RsvpStatus: rsvpStatusToProto(p.RSVPStatus),
	}
	if p.FSMemberID != nil {
		proto.FsMemberId = *p.FSMemberID
	}
	if p.Username != nil {
		proto.Username = *p.Username
	}
	if p.DisplayName != nil {
		proto.DisplayName = *p.DisplayName
	}
	if p.AvatarURL != nil {
		proto.AvatarUrl = *p.AvatarURL
	}
	if p.JoinedAt != nil {
		proto.JoinedAt = timestamppb.New(*p.JoinedAt)
	}
	if p.LeftAt != nil {
		proto.LeftAt = timestamppb.New(*p.LeftAt)
	}
	if p.RSVPAt != nil {
		proto.RsvpAt = timestamppb.New(*p.RSVPAt)
	}
	return proto
}

func participantStatusToProto(status model.ParticipantStatus) pb.ParticipantStatus {
	switch status {
	case model.ParticipantStatusConnecting:
		return pb.ParticipantStatus_PARTICIPANT_STATUS_CONNECTING
	case model.ParticipantStatusJoined:
		return pb.ParticipantStatus_PARTICIPANT_STATUS_JOINED
	case model.ParticipantStatusLeft:
		return pb.ParticipantStatus_PARTICIPANT_STATUS_LEFT
	case model.ParticipantStatusKicked:
		return pb.ParticipantStatus_PARTICIPANT_STATUS_KICKED
	default:
		return pb.ParticipantStatus_PARTICIPANT_STATUS_UNSPECIFIED
	}
}

func callToProto(call *model.Call) *pb.Call {
	proto := &pb.Call{
		Id:       call.ID.String(),
		CallerId: call.CallerID.String(),
		CalleeId: call.CalleeID.String(),
		Status:   callStatusToProto(call.Status),
		Duration: int32(call.Duration),
	}
	if call.ChatID != nil {
		proto.ChatId = call.ChatID.String()
	}
	if call.ConferenceID != nil {
		proto.ConferenceId = call.ConferenceID.String()
	}
	if call.FSCallUUID != nil {
		proto.FsCallUuid = *call.FSCallUUID
	}
	if call.EndReason != nil {
		proto.EndReason = *call.EndReason
	}
	if call.CallerUsername != nil {
		proto.CallerUsername = *call.CallerUsername
	}
	if call.CallerDisplayName != nil {
		proto.CallerDisplayName = *call.CallerDisplayName
	}
	if call.CalleeUsername != nil {
		proto.CalleeUsername = *call.CalleeUsername
	}
	if call.CalleeDisplayName != nil {
		proto.CalleeDisplayName = *call.CalleeDisplayName
	}
	if call.StartedAt != nil {
		proto.StartedAt = timestamppb.New(*call.StartedAt)
	}
	if call.AnsweredAt != nil {
		proto.AnsweredAt = timestamppb.New(*call.AnsweredAt)
	}
	if call.EndedAt != nil {
		proto.EndedAt = timestamppb.New(*call.EndedAt)
	}
	proto.CreatedAt = timestamppb.New(call.CreatedAt)
	return proto
}

func callStatusToProto(status model.CallStatus) pb.CallStatus {
	switch status {
	case model.CallStatusInitiated:
		return pb.CallStatus_CALL_STATUS_INITIATED
	case model.CallStatusRinging:
		return pb.CallStatus_CALL_STATUS_RINGING
	case model.CallStatusAnswered:
		return pb.CallStatus_CALL_STATUS_ANSWERED
	case model.CallStatusEnded:
		return pb.CallStatus_CALL_STATUS_ENDED
	case model.CallStatusFailed:
		return pb.CallStatus_CALL_STATUS_FAILED
	case model.CallStatusMissed:
		return pb.CallStatus_CALL_STATUS_MISSED
	default:
		return pb.CallStatus_CALL_STATUS_UNSPECIFIED
	}
}

func vertoCredentialsToProto(creds *model.VertoCredentials) *pb.VertoCredentials {
	iceServers := make([]*pb.IceServer, len(creds.IceServers))
	for i, ice := range creds.IceServers {
		iceServers[i] = &pb.IceServer{
			Urls:       ice.URLs,
			Username:   ice.Username,
			Credential: ice.Credential,
		}
	}

	return &pb.VertoCredentials{
		UserId:     creds.UserID.String(),
		Login:      creds.Login,
		Password:   creds.Password,
		WsUrl:      creds.WSUrl,
		IceServers: iceServers,
		ExpiresAt:  creds.ExpiresAt,
	}
}

// ======== Scheduled Events Helpers ========

func eventTypeToProto(et model.EventType) pb.EventType {
	switch et {
	case model.EventTypeAdhoc:
		return pb.EventType_EVENT_TYPE_ADHOC
	case model.EventTypeAdhocChat:
		return pb.EventType_EVENT_TYPE_ADHOC_CHAT
	case model.EventTypeScheduled:
		return pb.EventType_EVENT_TYPE_SCHEDULED
	case model.EventTypeRecurring:
		return pb.EventType_EVENT_TYPE_RECURRING
	default:
		return pb.EventType_EVENT_TYPE_UNSPECIFIED
	}
}

func conferenceRoleToProto(role model.ConferenceRole) pb.ConferenceRole {
	switch role {
	case model.RoleOriginator:
		return pb.ConferenceRole_CONFERENCE_ROLE_ORIGINATOR
	case model.RoleModerator:
		return pb.ConferenceRole_CONFERENCE_ROLE_MODERATOR
	case model.RoleSpeaker:
		return pb.ConferenceRole_CONFERENCE_ROLE_SPEAKER
	case model.RoleAssistant:
		return pb.ConferenceRole_CONFERENCE_ROLE_ASSISTANT
	case model.RoleParticipant:
		return pb.ConferenceRole_CONFERENCE_ROLE_PARTICIPANT
	default:
		return pb.ConferenceRole_CONFERENCE_ROLE_UNSPECIFIED
	}
}

func conferenceRoleFromProto(role pb.ConferenceRole) model.ConferenceRole {
	switch role {
	case pb.ConferenceRole_CONFERENCE_ROLE_ORIGINATOR:
		return model.RoleOriginator
	case pb.ConferenceRole_CONFERENCE_ROLE_MODERATOR:
		return model.RoleModerator
	case pb.ConferenceRole_CONFERENCE_ROLE_SPEAKER:
		return model.RoleSpeaker
	case pb.ConferenceRole_CONFERENCE_ROLE_ASSISTANT:
		return model.RoleAssistant
	case pb.ConferenceRole_CONFERENCE_ROLE_PARTICIPANT:
		return model.RoleParticipant
	default:
		return model.RoleParticipant
	}
}

func rsvpStatusToProto(status model.RSVPStatus) pb.RSVPStatus {
	switch status {
	case model.RSVPPending:
		return pb.RSVPStatus_RSVP_STATUS_PENDING
	case model.RSVPAccepted:
		return pb.RSVPStatus_RSVP_STATUS_ACCEPTED
	case model.RSVPDeclined:
		return pb.RSVPStatus_RSVP_STATUS_DECLINED
	default:
		return pb.RSVPStatus_RSVP_STATUS_UNSPECIFIED
	}
}

func rsvpStatusFromProto(status pb.RSVPStatus) model.RSVPStatus {
	switch status {
	case pb.RSVPStatus_RSVP_STATUS_PENDING:
		return model.RSVPPending
	case pb.RSVPStatus_RSVP_STATUS_ACCEPTED:
		return model.RSVPAccepted
	case pb.RSVPStatus_RSVP_STATUS_DECLINED:
		return model.RSVPDeclined
	default:
		return model.RSVPPending
	}
}

func recurrenceRuleToProto(rule *model.RecurrenceRule) *pb.RecurrenceRule {
	if rule == nil {
		return nil
	}

	proto := &pb.RecurrenceRule{
		Frequency:   recurrenceFrequencyToProto(rule.Frequency),
		DaysOfWeek:  make([]int32, len(rule.DaysOfWeek)),
	}

	for i, d := range rule.DaysOfWeek {
		proto.DaysOfWeek[i] = int32(d)
	}

	if rule.DayOfMonth != nil {
		proto.DayOfMonth = int32(*rule.DayOfMonth)
	}

	if rule.UntilDate != nil {
		proto.Until = timestamppb.New(*rule.UntilDate)
	}

	if rule.OccurrenceCount != nil {
		proto.Count = int32(*rule.OccurrenceCount)
	}

	return proto
}

func recurrenceRuleFromProto(proto *pb.RecurrenceRule) *model.RecurrenceRule {
	if proto == nil {
		return nil
	}

	rule := &model.RecurrenceRule{
		Frequency:  recurrenceFrequencyFromProto(proto.GetFrequency()),
		DaysOfWeek: make([]int, len(proto.GetDaysOfWeek())),
	}

	for i, d := range proto.GetDaysOfWeek() {
		rule.DaysOfWeek[i] = int(d)
	}

	if proto.GetDayOfMonth() != 0 {
		dayOfMonth := int(proto.GetDayOfMonth())
		rule.DayOfMonth = &dayOfMonth
	}

	if proto.GetUntil() != nil {
		until := proto.GetUntil().AsTime()
		rule.UntilDate = &until
	}

	if proto.GetCount() != 0 {
		count := int(proto.GetCount())
		rule.OccurrenceCount = &count
	}

	return rule
}

func recurrenceFrequencyToProto(freq model.RecurrenceFrequency) pb.RecurrenceFrequency {
	switch freq {
	case model.RecurrenceDaily:
		return pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_DAILY
	case model.RecurrenceWeekly:
		return pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_WEEKLY
	case model.RecurrenceBiweekly:
		return pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_BIWEEKLY
	case model.RecurrenceMonthly:
		return pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_MONTHLY
	default:
		return pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_UNSPECIFIED
	}
}

func recurrenceFrequencyFromProto(freq pb.RecurrenceFrequency) model.RecurrenceFrequency {
	switch freq {
	case pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_DAILY:
		return model.RecurrenceDaily
	case pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_WEEKLY:
		return model.RecurrenceWeekly
	case pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_BIWEEKLY:
		return model.RecurrenceBiweekly
	case pb.RecurrenceFrequency_RECURRENCE_FREQUENCY_MONTHLY:
		return model.RecurrenceMonthly
	default:
		return model.RecurrenceWeekly
	}
}
