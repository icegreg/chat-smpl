package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/icegreg/chat-smpl/proto/files"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
	"github.com/icegreg/chat-smpl/services/files/internal/service"
)

// FilesServer implements the gRPC FilesService interface
type FilesServer struct {
	pb.UnimplementedFilesServiceServer
	fileService service.FileService
}

func NewFilesServer(fileService service.FileService) *FilesServer {
	return &FilesServer{
		fileService: fileService,
	}
}

// Helper functions

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func parseUUIDs(strs []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func handleError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, repository.ErrFileNotFound):
		return status.Error(codes.NotFound, "file not found")
	case errors.Is(err, repository.ErrFileLinkNotFound):
		return status.Error(codes.NotFound, "file link not found")
	case errors.Is(err, repository.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, "access denied")
	case errors.Is(err, service.ErrFileNotReadable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, service.ErrFileIsDirectory):
		return status.Error(codes.InvalidArgument, "path is a directory")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// AddLocalFile adds a file from the server's local filesystem
func (s *FilesServer) AddLocalFile(ctx context.Context, req *pb.AddLocalFileRequest) (*pb.AddLocalFileResponse, error) {
	uploadedBy, err := parseUUID(req.UploadedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid uploaded_by ID")
	}

	result, err := s.fileService.AddLocalFile(ctx, req.ServerFilePath, req.OriginalFilename, req.ContentType, uploadedBy)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.AddLocalFileResponse{
		FileId:           result.ID.String(),
		LinkId:           result.LinkID.String(),
		Filename:         result.Filename,
		OriginalFilename: result.OriginalFilename,
		ContentType:      result.ContentType,
		Size:             result.Size,
	}, nil
}

// CreateFileLink creates a new link for an existing file (for message forwarding)
func (s *FilesServer) CreateFileLink(ctx context.Context, req *pb.CreateFileLinkRequest) (*pb.CreateFileLinkResponse, error) {
	fileID, err := parseUUID(req.FileId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_id")
	}

	createdBy, err := parseUUID(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by ID")
	}

	linkID, err := s.fileService.CreateFileLink(ctx, fileID, createdBy)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateFileLinkResponse{
		LinkId: linkID.String(),
		FileId: fileID.String(),
	}, nil
}

// GrantPermissions grants view/download permissions to users for file links
func (s *FilesServer) GrantPermissions(ctx context.Context, req *pb.GrantPermissionsRequest) (*emptypb.Empty, error) {
	linkIDs, err := parseUUIDs(req.LinkIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link_id")
	}

	userIDs, err := parseUUIDs(req.UserIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	granterID, err := parseUUID(req.GranterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid granter_id")
	}

	if err := s.fileService.GrantPermissions(ctx, linkIDs, userIDs, granterID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// RevokePermissions revokes permissions from a user for file links
func (s *FilesServer) RevokePermissions(ctx context.Context, req *pb.RevokePermissionsRequest) (*emptypb.Empty, error) {
	linkIDs, err := parseUUIDs(req.LinkIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link_id")
	}

	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.fileService.RevokePermissions(ctx, linkIDs, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// GetFileIDByLinkID returns the file ID for a given link ID
func (s *FilesServer) GetFileIDByLinkID(ctx context.Context, req *pb.GetFileIDByLinkIDRequest) (*pb.GetFileIDByLinkIDResponse, error) {
	linkID, err := parseUUID(req.LinkId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link_id")
	}

	fileID, err := s.fileService.GetFileIDByLinkID(ctx, linkID)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.GetFileIDByLinkIDResponse{
		FileId: fileID.String(),
	}, nil
}

// GetFilesByLinkIDs returns file metadata for multiple link IDs
func (s *FilesServer) GetFilesByLinkIDs(ctx context.Context, req *pb.GetFilesByLinkIDsRequest) (*pb.GetFilesByLinkIDsResponse, error) {
	linkIDs, err := parseUUIDs(req.LinkIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link_id")
	}

	files, err := s.fileService.GetFilesByLinkIDs(ctx, linkIDs)
	if err != nil {
		return nil, handleError(err)
	}

	protoFiles := make([]*pb.FileAttachment, 0, len(files))
	for _, f := range files {
		protoFiles = append(protoFiles, &pb.FileAttachment{
			LinkId:           f.LinkID.String(),
			FileId:           f.ID.String(),
			Filename:         f.Filename,
			OriginalFilename: f.OriginalFilename,
			ContentType:      f.ContentType,
			Size:             f.Size,
		})
	}

	return &pb.GetFilesByLinkIDsResponse{
		Files: protoFiles,
	}, nil
}

// ============ File Groups Management ============

// CreateFileGroup creates a new file group with permissions
func (s *FilesServer) CreateFileGroup(ctx context.Context, req *pb.CreateFileGroupRequest) (*pb.CreateFileGroupResponse, error) {
	group, err := s.fileService.CreateFileGroup(ctx, req.Name, req.CanRead, req.CanDelete, req.CanTransfer)
	if err != nil {
		return nil, handleError(err)
	}

	return &pb.CreateFileGroupResponse{
		Group: &pb.FileGroup{
			Id:          group.ID.String(),
			Name:        group.Name,
			CanRead:     group.CanRead,
			CanDelete:   group.CanDelete,
			CanTransfer: group.CanTransfer,
		},
	}, nil
}

// DeleteFileGroup deletes a file group
func (s *FilesServer) DeleteFileGroup(ctx context.Context, req *pb.DeleteFileGroupRequest) (*emptypb.Empty, error) {
	groupID, err := parseUUID(req.GroupId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	if err := s.fileService.DeleteFileGroup(ctx, groupID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// AddUserToGroup adds a user to a file group
func (s *FilesServer) AddUserToGroup(ctx context.Context, req *pb.AddUserToGroupRequest) (*emptypb.Empty, error) {
	groupID, err := parseUUID(req.GroupId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.fileService.AddUserToGroup(ctx, groupID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// RemoveUserFromGroup removes a user from a file group
func (s *FilesServer) RemoveUserFromGroup(ctx context.Context, req *pb.RemoveUserFromGroupRequest) (*emptypb.Empty, error) {
	groupID, err := parseUUID(req.GroupId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.fileService.RemoveUserFromGroup(ctx, groupID, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// AddFileLinkToGroups associates a file link with multiple groups
func (s *FilesServer) AddFileLinkToGroups(ctx context.Context, req *pb.AddFileLinkToGroupsRequest) (*emptypb.Empty, error) {
	fileLinkID, err := parseUUID(req.FileLinkId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file_link_id")
	}

	groupIDs, err := parseUUIDs(req.GroupIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	if err := s.fileService.AddFileLinkToGroups(ctx, fileLinkID, groupIDs); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}

// GetFilesByGroup returns all file links in a group
func (s *FilesServer) GetFilesByGroup(ctx context.Context, req *pb.GetFilesByGroupRequest) (*pb.GetFilesByGroupResponse, error) {
	groupID, err := parseUUID(req.GroupId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	links, err := s.fileService.GetFilesByGroup(ctx, groupID)
	if err != nil {
		return nil, handleError(err)
	}

	protoLinks := make([]*pb.FileLinkInfo, 0, len(links))
	for _, link := range links {
		protoLinks = append(protoLinks, &pb.FileLinkInfo{
			Id:         link.ID.String(),
			FileId:     link.FileID.String(),
			UploadedBy: link.UploadedBy.String(),
		})
	}

	return &pb.GetFilesByGroupResponse{
		FileLinks: protoLinks,
	}, nil
}

// RemoveUserFromAllGroupFiles removes user from multiple groups and revokes permissions
func (s *FilesServer) RemoveUserFromAllGroupFiles(ctx context.Context, req *pb.RemoveUserFromAllGroupFilesRequest) (*emptypb.Empty, error) {
	groupIDs, err := parseUUIDs(req.GroupIds)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid group_id")
	}

	userID, err := parseUUID(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.fileService.RemoveUserFromAllGroupFiles(ctx, groupIDs, userID); err != nil {
		return nil, handleError(err)
	}

	return &emptypb.Empty{}, nil
}
