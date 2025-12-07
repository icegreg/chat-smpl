package handler

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/icegreg/chat-smpl/proto/presence"
	"github.com/icegreg/chat-smpl/services/presence/internal/repository"
	"github.com/icegreg/chat-smpl/services/presence/internal/service"
)

// GRPCHandler implements the PresenceService gRPC server
type GRPCHandler struct {
	pb.UnimplementedPresenceServiceServer
	service *service.Service
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(svc *service.Service) *GRPCHandler {
	return &GRPCHandler{service: svc}
}

// SetStatus sets user's status
func (h *GRPCHandler) SetStatus(ctx context.Context, req *pb.SetStatusRequest) (*pb.SetStatusResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	repoStatus := protoStatusToRepo(req.Status)
	presence, err := h.service.SetStatus(ctx, req.UserId, repoStatus)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set status: %v", err)
	}

	return &pb.SetStatusResponse{
		Presence: presenceToProto(presence),
	}, nil
}

// GetPresence gets presence info for a single user
func (h *GRPCHandler) GetPresence(ctx context.Context, req *pb.GetPresenceRequest) (*pb.GetPresenceResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	presence, err := h.service.GetPresence(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get presence: %v", err)
	}

	return &pb.GetPresenceResponse{
		Presence: presenceToProto(presence),
	}, nil
}

// GetPresencesBatch gets presence info for multiple users
func (h *GRPCHandler) GetPresencesBatch(ctx context.Context, req *pb.GetPresencesBatchRequest) (*pb.GetPresencesBatchResponse, error) {
	if len(req.UserIds) == 0 {
		return &pb.GetPresencesBatchResponse{}, nil
	}

	presences, err := h.service.GetPresencesBatch(ctx, req.UserIds)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get presences: %v", err)
	}

	protoPresences := make([]*pb.PresenceInfo, 0, len(presences))
	for _, p := range presences {
		protoPresences = append(protoPresences, presenceToProto(p))
	}

	return &pb.GetPresencesBatchResponse{
		Presences: protoPresences,
	}, nil
}

// UserConnected is called when user establishes websocket connection
func (h *GRPCHandler) UserConnected(ctx context.Context, req *pb.UserConnectedRequest) (*pb.UserConnectedResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.ConnectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "connection_id is required")
	}

	presence, err := h.service.UserConnected(ctx, req.UserId, req.ConnectionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record connection: %v", err)
	}

	return &pb.UserConnectedResponse{
		Presence: presenceToProto(presence),
	}, nil
}

// UserDisconnected is called when user's websocket connection closes
func (h *GRPCHandler) UserDisconnected(ctx context.Context, req *pb.UserDisconnectedRequest) (*pb.UserDisconnectedResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.ConnectionId == "" {
		return nil, status.Error(codes.InvalidArgument, "connection_id is required")
	}

	presence, err := h.service.UserDisconnected(ctx, req.UserId, req.ConnectionId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record disconnection: %v", err)
	}

	return &pb.UserDisconnectedResponse{
		Presence: presenceToProto(presence),
	}, nil
}

// Helper functions

func protoStatusToRepo(s pb.UserStatus) repository.UserStatus {
	switch s {
	case pb.UserStatus_STATUS_BUSY:
		return repository.StatusBusy
	case pb.UserStatus_STATUS_AWAY:
		return repository.StatusAway
	case pb.UserStatus_STATUS_DND:
		return repository.StatusDND
	default:
		return repository.StatusAvailable
	}
}

func repoStatusToProto(s repository.UserStatus) pb.UserStatus {
	switch s {
	case repository.StatusBusy:
		return pb.UserStatus_STATUS_BUSY
	case repository.StatusAway:
		return pb.UserStatus_STATUS_AWAY
	case repository.StatusDND:
		return pb.UserStatus_STATUS_DND
	default:
		return pb.UserStatus_STATUS_AVAILABLE
	}
}

func presenceToProto(p *repository.PresenceInfo) *pb.PresenceInfo {
	lastSeenAt := ""
	if !p.LastSeenAt.IsZero() {
		lastSeenAt = p.LastSeenAt.Format(time.RFC3339)
	}

	return &pb.PresenceInfo{
		UserId:          p.UserID,
		Status:          repoStatusToProto(p.Status),
		IsOnline:        p.IsOnline,
		ConnectionCount: int32(p.ConnectionCount),
		LastSeenAt:      lastSeenAt,
	}
}
