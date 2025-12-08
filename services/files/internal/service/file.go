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
	Upload(ctx context.Context, filename, contentType string, size int64, reader io.Reader, uploadedBy uuid.UUID) (*model.UploadFileResponse, error)
	Download(ctx context.Context, fileLinkID, userID uuid.UUID) (io.ReadCloser, *model.File, error)
	DownloadByShareToken(ctx context.Context, token, password string) (io.ReadCloser, *model.File, error)
	Delete(ctx context.Context, fileLinkID, userID uuid.UUID) error
	CreateShareLink(ctx context.Context, fileID, userID uuid.UUID, req model.CreateShareLinkRequest) (*model.ShareLinkDTO, error)
	GetFileInfo(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileDTO, error)
	GetAvatar(ctx context.Context, userID string) (io.ReadCloser, string, error)
	GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) ([]model.FileAttachmentDTO, error)
	GrantPermissions(ctx context.Context, linkIDs []uuid.UUID, userIDs []uuid.UUID, uploaderID uuid.UUID) error

	// gRPC methods for inter-service communication
	AddLocalFile(ctx context.Context, serverPath, originalFilename, contentType string, uploadedBy uuid.UUID) (*model.UploadFileResponse, error)
	CreateFileLink(ctx context.Context, fileID, createdBy uuid.UUID) (uuid.UUID, error)
	RevokePermissions(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error
	GetFileIDByLinkID(ctx context.Context, linkID uuid.UUID) (uuid.UUID, error)
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

	// Check permission
	perm, err := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
	if err != nil {
		return nil, nil, err
	}

	if !perm.CanDownload {
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

	// Check permission
	perm, err := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
	if err != nil {
		return err
	}

	if !perm.CanDelete {
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

	// Check permission
	perm, err := s.repo.GetFileLinkPermission(ctx, fileLinkID, userID)
	if err != nil {
		return nil, err
	}

	if !perm.CanView {
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
		if err := s.repo.CreatePermissionsForParticipants(ctx, linkID, userIDs, uploaderID); err != nil {
			return fmt.Errorf("failed to create permissions for link %s: %w", linkID, err)
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
