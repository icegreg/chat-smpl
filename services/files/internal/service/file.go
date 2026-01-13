package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/icegreg/chat-smpl/services/files/internal/model"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
	"github.com/icegreg/chat-smpl/services/files/internal/storage"
)

var (
	ErrFileNotReadable = errors.New("file not readable")
	ErrFileIsDirectory = errors.New("path is a directory")
)

type FileService interface {
	// Core file operations
	Upload(ctx context.Context, filename, contentType string, size int64, reader io.Reader, uploadedBy uuid.UUID) (*model.UploadFileResponse, error)
	Download(ctx context.Context, fileLinkID, userID uuid.UUID) (io.ReadCloser, *model.File, error)
	DownloadByShareToken(ctx context.Context, token, password string) (io.ReadCloser, *model.File, error)
	Delete(ctx context.Context, fileLinkID, userID uuid.UUID) error
	CreateShareLink(ctx context.Context, fileID, userID uuid.UUID, req model.CreateShareLinkRequest) (*model.ShareLinkDTO, error)
	GetFileInfo(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileDTO, error)
	GetAvatar(ctx context.Context, userID string) (io.ReadCloser, string, error)
	GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) ([]model.FileAttachmentDTO, error)

	// Individual permissions (for standalone files)
	GrantPermissions(ctx context.Context, linkIDs []uuid.UUID, userIDs []uuid.UUID, uploaderID uuid.UUID) error

	// gRPC methods for inter-service communication
	AddLocalFile(ctx context.Context, serverPath, originalFilename, contentType string, uploadedBy uuid.UUID) (*model.UploadFileResponse, error)
	CreateFileLink(ctx context.Context, fileID, createdBy uuid.UUID) (uuid.UUID, error)
	RevokePermissions(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error
	GetFileIDByLinkID(ctx context.Context, linkID uuid.UUID) (uuid.UUID, error)

	// File groups management (Chat Service calls these)
	CreateFileGroup(ctx context.Context, name string, canRead, canDelete, canTransfer bool) (*model.FileGroup, error)
	DeleteFileGroup(ctx context.Context, groupID uuid.UUID) error
	AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error
	RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error
	AddFileLinkToGroups(ctx context.Context, fileLinkID uuid.UUID, groupIDs []uuid.UUID) error
	GetFilesByGroup(ctx context.Context, groupID uuid.UUID) ([]model.FileLink, error)
	RemoveUserFromAllGroupFiles(ctx context.Context, groupIDs []uuid.UUID, userID uuid.UUID) error

	// Chat files
	GetChatFiles(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.ChatFileDTO, int, error)
}

type fileService struct {
	repo    repository.FileRepository
	storage storage.Storage
	baseURL string
}

func NewFileService(repo repository.FileRepository, storage storage.Storage, baseURL string) FileService {
	return &fileService{
		repo:    repo,
		storage: storage,
		baseURL: baseURL,
	}
}

func (s *fileService) Upload(ctx context.Context, filename, contentType string, size int64, reader io.Reader, uploadedBy uuid.UUID) (*model.UploadFileResponse, error) {
	// Save file to storage
	storagePath, err := s.storage.Save(filename, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Create file record
	file := &model.File{
		Filename:         filepath.Base(storagePath),
		OriginalFilename: filename,
		ContentType:      contentType,
		Size:             size,
		FilePath:         storagePath,
		UploadedBy:       uploadedBy,
		Status:           model.FileStatusActive,
	}

	if err := s.repo.CreateFile(ctx, file); err != nil {
		s.storage.Delete(storagePath)
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Create file link
	link := &model.FileLink{
		FileID:     file.ID,
		UploadedBy: uploadedBy,
	}

	if err := s.repo.CreateFileLink(ctx, link); err != nil {
		s.storage.Delete(storagePath)
		return nil, fmt.Errorf("failed to create file link: %w", err)
	}

	// Create permission for uploader
	perm := &model.FileLinkPermission{
		FileLinkID:  link.ID,
		UserID:      uploadedBy,
		CanView:     true,
		CanDownload: true,
		CanDelete:   true,
	}

	if err := s.repo.CreateFileLinkPermission(ctx, perm); err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return &model.UploadFileResponse{
		ID:               file.ID,
		LinkID:           link.ID,
		Filename:         file.Filename,
		OriginalFilename: file.OriginalFilename,
		ContentType:      file.ContentType,
		Size:             file.Size,
	}, nil
}

func (s *fileService) Download(ctx context.Context, fileLinkID, userID uuid.UUID) (io.ReadCloser, *model.File, error) {
	// Get file link
	link, err := s.repo.GetFileLink(ctx, fileLinkID)
	if err != nil {
		return nil, nil, err
	}

	if link.IsDeleted {
		return nil, nil, repository.ErrFileLinkNotFound
	}

	// Check permission using the group-based access check
	accessLevel, err := s.repo.CheckFileAccess(ctx, fileLinkID, userID)
	if err != nil {
		// Fall back to individual permissions if check_file_access function is not available
		perm, permErr := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
		if permErr != nil {
			return nil, nil, permErr
		}
		if !perm.CanDownload {
			return nil, nil, repository.ErrAccessDenied
		}
	} else if accessLevel == model.FileAccessNone {
		return nil, nil, repository.ErrAccessDenied
	}

	// Get file
	file, err := s.repo.GetFile(ctx, link.FileID)
	if err != nil {
		return nil, nil, err
	}

	// Get file from storage
	reader, err := s.storage.Get(file.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file from storage: %w", err)
	}

	return reader, file, nil
}

func (s *fileService) DownloadByShareToken(ctx context.Context, token, password string) (io.ReadCloser, *model.File, error) {
	// Get share link
	shareLink, err := s.repo.GetShareLinkByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	// Check password if set
	if shareLink.Password != nil && *shareLink.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*shareLink.Password), []byte(password)); err != nil {
			return nil, nil, repository.ErrAccessDenied
		}
	}

	// Get file
	file, err := s.repo.GetFile(ctx, shareLink.FileID)
	if err != nil {
		return nil, nil, err
	}

	// Increment download count
	if err := s.repo.IncrementDownloadCount(ctx, shareLink.ID); err != nil {
		return nil, nil, err
	}

	// Get file from storage
	reader, err := s.storage.Get(file.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file from storage: %w", err)
	}

	return reader, file, nil
}

func (s *fileService) Delete(ctx context.Context, fileLinkID, userID uuid.UUID) error {
	// Get file link
	link, err := s.repo.GetFileLink(ctx, fileLinkID)
	if err != nil {
		return err
	}

	// Check permission using the group-based access check
	accessLevel, err := s.repo.CheckFileAccess(ctx, fileLinkID, userID)
	if err != nil {
		// Fall back to individual permissions if check_file_access function is not available
		perm, permErr := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
		if permErr != nil {
			return permErr
		}
		if !perm.CanDelete {
			return repository.ErrAccessDenied
		}
	} else if accessLevel != model.FileAccessDelete && accessLevel != model.FileAccessTransfer {
		return repository.ErrAccessDenied
	}

	// Soft delete file link
	return s.repo.SoftDeleteFileLink(ctx, link.ID)
}

func (s *fileService) CreateShareLink(ctx context.Context, fileID, userID uuid.UUID, req model.CreateShareLinkRequest) (*model.ShareLinkDTO, error) {
	// Check if user owns the file or has permission
	link, err := s.repo.GetFileLinkByFileID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	perm, err := s.repo.GetFileLinkPermission(ctx, link.ID, userID)
	if err != nil {
		return nil, err
	}

	if !perm.CanView {
		return nil, repository.ErrAccessDenied
	}

	// Hash password if provided
	var hashedPassword *string
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashStr := string(hash)
		hashedPassword = &hashStr
	}

	shareLink := &model.FileShareLink{
		FileID:       fileID,
		Password:     hashedPassword,
		MaxDownloads: req.MaxDownloads,
		ExpiresAt:    req.ExpiresAt,
		CreatedBy:    userID,
	}

	if err := s.repo.CreateShareLink(ctx, shareLink); err != nil {
		return nil, err
	}

	dto := shareLink.ToDTO(s.baseURL)
	return &dto, nil
}

func (s *fileService) GetFileInfo(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileDTO, error) {
	// Get file link
	link, err := s.repo.GetFileLink(ctx, fileLinkID)
	if err != nil {
		return nil, err
	}

	if link.IsDeleted {
		return nil, repository.ErrFileLinkNotFound
	}

	// Check permission using the new group-based access check
	accessLevel, err := s.repo.CheckFileAccess(ctx, fileLinkID, userID)
	if err != nil {
		// Fall back to individual permissions if check_file_access function is not available
		perm, permErr := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
		if permErr != nil {
			return nil, permErr
		}
		if !perm.CanView {
			return nil, repository.ErrAccessDenied
		}
	} else if accessLevel == model.FileAccessNone {
		return nil, repository.ErrAccessDenied
	}

	// Get file
	file, err := s.repo.GetFile(ctx, link.FileID)
	if err != nil {
		return nil, err
	}

	dto := file.ToDTO()
	return &dto, nil
}

func (s *fileService) GetAvatar(ctx context.Context, userID string) (io.ReadCloser, string, error) {
	// Avatars are stored in avatars/{userID}.jpg
	avatarPath := filepath.Join("avatars", userID+".jpg")

	if !s.storage.Exists(avatarPath) {
		return nil, "", fmt.Errorf("avatar not found")
	}

	reader, err := s.storage.Get(avatarPath)
	if err != nil {
		return nil, "", err
	}

	return reader, "image/jpeg", nil
}

func (s *fileService) GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) ([]model.FileAttachmentDTO, error) {
	files, err := s.repo.GetFilesByLinkIDs(ctx, linkIDs)
	if err != nil {
		return nil, err
	}

	result := make([]model.FileAttachmentDTO, 0, len(files))
	for linkID, file := range files {
		result = append(result, model.FileAttachmentDTO{
			LinkID:           linkID,
			ID:               file.ID,
			Filename:         file.Filename,
			OriginalFilename: file.OriginalFilename,
			ContentType:      file.ContentType,
			Size:             file.Size,
		})
	}

	return result, nil
}

// GrantPermissions grants individual permissions on files to multiple users
func (s *fileService) GrantPermissions(ctx context.Context, linkIDs []uuid.UUID, userIDs []uuid.UUID, uploaderID uuid.UUID) error {
	for _, linkID := range linkIDs {
		// Verify the uploader owns this file link
		link, err := s.repo.GetFileLink(ctx, linkID)
		if err != nil {
			continue // Skip if file link not found
		}

		// Only the uploader can grant permissions
		if link.UploadedBy != uploaderID {
			continue
		}

		// Create permissions for all users
		for _, userID := range userIDs {
			perm := &model.FileLinkPermission{
				FileLinkID:  linkID,
				UserID:      userID,
				CanView:     true,
				CanDownload: true,
				CanDelete:   userID == uploaderID,
			}
			if err := s.repo.CreateFileLinkPermission(ctx, perm); err != nil {
				return fmt.Errorf("failed to create permission for link %s, user %s: %w", linkID, userID, err)
			}
		}
	}

	return nil
}


// AddLocalFile adds a file from the server's local filesystem
func (s *fileService) AddLocalFile(ctx context.Context, serverPath, originalFilename, contentType string, uploadedBy uuid.UUID) (*model.UploadFileResponse, error) {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(serverPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrFileNotReadable, serverPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if fileInfo.IsDir() {
		return nil, ErrFileIsDirectory
	}

	// Open the file
	file, err := os.Open(serverPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFileNotReadable, err.Error())
	}
	defer file.Close()

	// Detect content type if not provided
	if contentType == "" {
		ext := filepath.Ext(serverPath)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	// Use original filename if not provided
	if originalFilename == "" {
		originalFilename = filepath.Base(serverPath)
	}

	// Use existing Upload method
	return s.Upload(ctx, originalFilename, contentType, fileInfo.Size(), file, uploadedBy)
}

// CreateFileLink creates a new link for an existing file (used for message forwarding)
func (s *fileService) CreateFileLink(ctx context.Context, fileID, createdBy uuid.UUID) (uuid.UUID, error) {
	// Verify file exists
	_, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return uuid.Nil, err
	}

	// Create new file link
	link := &model.FileLink{
		FileID:     fileID,
		UploadedBy: createdBy,
	}

	if err := s.repo.CreateFileLink(ctx, link); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create file link: %w", err)
	}

	// Create permission for the link creator
	perm := &model.FileLinkPermission{
		FileLinkID:  link.ID,
		UserID:      createdBy,
		CanView:     true,
		CanDownload: true,
		CanDelete:   true,
	}

	if err := s.repo.CreateFileLinkPermission(ctx, perm); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return link.ID, nil
}

// RevokePermissions removes permissions from a user for specified file links
func (s *fileService) RevokePermissions(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeletePermissionsForUser(ctx, linkIDs, userID)
}

// GetFileIDByLinkID returns the file ID for a given link ID
func (s *fileService) GetFileIDByLinkID(ctx context.Context, linkID uuid.UUID) (uuid.UUID, error) {
	link, err := s.repo.GetFileLink(ctx, linkID)
	if err != nil {
		return uuid.Nil, err
	}
	return link.FileID, nil
}

// CreateFileGroup creates a new file group with specified permissions
func (s *fileService) CreateFileGroup(ctx context.Context, name string, canRead, canDelete, canTransfer bool) (*model.FileGroup, error) {
	group := &model.FileGroup{
		Name:        name,
		CanRead:     canRead,
		CanDelete:   canDelete,
		CanTransfer: canTransfer,
	}

	if err := s.repo.CreateFileGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create file group: %w", err)
	}

	return group, nil
}

// DeleteFileGroup deletes a file group and all its associations
func (s *fileService) DeleteFileGroup(ctx context.Context, groupID uuid.UUID) error {
	return s.repo.DeleteFileGroup(ctx, groupID)
}

// AddUserToGroup adds a user to a file group
func (s *fileService) AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.repo.AddUserToGroup(ctx, groupID, userID)
}

// RemoveUserFromGroup removes a user from a file group
func (s *fileService) RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.repo.RemoveUserFromGroup(ctx, groupID, userID)
}

// AddFileLinkToGroups associates a file link with multiple groups
func (s *fileService) AddFileLinkToGroups(ctx context.Context, fileLinkID uuid.UUID, groupIDs []uuid.UUID) error {
	for _, groupID := range groupIDs {
		if err := s.repo.AddFileLinkToGroup(ctx, fileLinkID, groupID); err != nil {
			return fmt.Errorf("failed to add file link to group %s: %w", groupID, err)
		}
	}
	return nil
}

// GetFilesByGroup returns all file links associated with a group
func (s *fileService) GetFilesByGroup(ctx context.Context, groupID uuid.UUID) ([]model.FileLink, error) {
	return s.repo.GetFilesByGroup(ctx, groupID)
}

// RemoveUserFromAllGroupFiles removes user from groups and revokes individual permissions on group files
func (s *fileService) RemoveUserFromAllGroupFiles(ctx context.Context, groupIDs []uuid.UUID, userID uuid.UUID) error {
	// First, collect all file link IDs from all groups
	var allLinkIDs []uuid.UUID
	for _, groupID := range groupIDs {
		links, err := s.repo.GetFilesByGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("failed to get files by group %s: %w", groupID, err)
		}
		for _, link := range links {
			allLinkIDs = append(allLinkIDs, link.ID)
		}

		// Remove user from the group
		if err := s.repo.RemoveUserFromGroup(ctx, groupID, userID); err != nil {
			return fmt.Errorf("failed to remove user from group %s: %w", groupID, err)
		}
	}

	// Revoke individual permissions on all files
	if len(allLinkIDs) > 0 {
		if err := s.repo.DeletePermissionsForUser(ctx, allLinkIDs, userID); err != nil {
			return fmt.Errorf("failed to revoke permissions: %w", err)
		}
	}

	return nil
}

// GetChatFiles returns files that were attached to messages in a specific chat
func (s *fileService) GetChatFiles(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.ChatFileDTO, int, error) {
	return s.repo.GetChatFiles(ctx, chatID, limit, offset)
}
