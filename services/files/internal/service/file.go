package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/icegreg/chat-smpl/services/files/internal/model"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
	"github.com/icegreg/chat-smpl/services/files/internal/storage"
)

type FileService interface {
	Upload(ctx context.Context, filename, contentType string, size int64, reader io.Reader, uploadedBy uuid.UUID) (*model.UploadFileResponse, error)
	Download(ctx context.Context, fileLinkID, userID uuid.UUID) (io.ReadCloser, *model.File, error)
	DownloadByShareToken(ctx context.Context, token, password string) (io.ReadCloser, *model.File, error)
	Delete(ctx context.Context, fileLinkID, userID uuid.UUID) error
	CreateShareLink(ctx context.Context, fileID, userID uuid.UUID, req model.CreateShareLinkRequest) (*model.ShareLinkDTO, error)
	GetFileInfo(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileDTO, error)
	GetAvatar(ctx context.Context, userID string) (io.ReadCloser, string, error)
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
