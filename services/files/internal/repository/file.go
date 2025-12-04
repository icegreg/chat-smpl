package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/services/files/internal/model"
)

var (
	ErrFileNotFound      = errors.New("file not found")
	ErrFileLinkNotFound  = errors.New("file link not found")
	ErrShareLinkNotFound = errors.New("share link not found")
	ErrShareLinkExpired  = errors.New("share link has expired")
	ErrShareLinkInactive = errors.New("share link is inactive")
	ErrMaxDownloads      = errors.New("maximum downloads reached")
	ErrAccessDenied      = errors.New("access denied")
)

type FileRepository interface {
	// File operations
	CreateFile(ctx context.Context, file *model.File) error
	GetFile(ctx context.Context, id uuid.UUID) (*model.File, error)
	UpdateFileStatus(ctx context.Context, id uuid.UUID, status model.FileStatus) error
	DeleteFile(ctx context.Context, id uuid.UUID) error

	// File link operations
	CreateFileLink(ctx context.Context, link *model.FileLink) error
	GetFileLink(ctx context.Context, id uuid.UUID) (*model.FileLink, error)
	GetFileLinkByFileID(ctx context.Context, fileID uuid.UUID) (*model.FileLink, error)
	DeleteFileLink(ctx context.Context, id uuid.UUID) error
	SoftDeleteFileLink(ctx context.Context, id uuid.UUID) error

	// Permissions
	CreateFileLinkPermission(ctx context.Context, perm *model.FileLinkPermission) error
	GetFileLinkPermission(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileLinkPermission, error)
	CreatePermissionsForParticipants(ctx context.Context, fileLinkID uuid.UUID, participantIDs []uuid.UUID, uploaderID uuid.UUID) error

	// Share links
	CreateShareLink(ctx context.Context, link *model.FileShareLink) error
	GetShareLinkByToken(ctx context.Context, token string) (*model.FileShareLink, error)
	IncrementDownloadCount(ctx context.Context, id uuid.UUID) error
	DeactivateShareLink(ctx context.Context, id uuid.UUID) error

	// Message attachments
	CreateMessageAttachment(ctx context.Context, attachment *model.MessageFileAttachment) error
	GetMessageAttachments(ctx context.Context, messageID uuid.UUID) ([]model.MessageFileAttachment, error)
}

type fileRepository struct {
	pool *pgxpool.Pool
}

func NewFileRepository(pool *pgxpool.Pool) FileRepository {
	return &fileRepository{pool: pool}
}

// File operations

func (r *fileRepository) CreateFile(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO con_test.files (id, filename, original_filename, content_type, size, file_path, uploaded_by, uploaded_at, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	file.ID = uuid.New()
	file.UploadedAt = time.Now()
	if file.Status == "" {
		file.Status = model.FileStatusActive
	}
	if file.Metadata == nil {
		file.Metadata = []byte("{}")
	}

	_, err := r.pool.Exec(ctx, query,
		file.ID, file.Filename, file.OriginalFilename, file.ContentType,
		file.Size, file.FilePath, file.UploadedBy, file.UploadedAt,
		file.Status, file.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

func (r *fileRepository) GetFile(ctx context.Context, id uuid.UUID) (*model.File, error) {
	query := `
		SELECT id, filename, original_filename, content_type, size, file_path, uploaded_by, uploaded_at, status, metadata
		FROM con_test.files
		WHERE id = $1
	`

	var file model.File
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&file.ID, &file.Filename, &file.OriginalFilename, &file.ContentType,
		&file.Size, &file.FilePath, &file.UploadedBy, &file.UploadedAt,
		&file.Status, &file.Metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return &file, nil
}

func (r *fileRepository) UpdateFileStatus(ctx context.Context, id uuid.UUID, status model.FileStatus) error {
	query := `UPDATE con_test.files SET status = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

func (r *fileRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM con_test.files WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrFileNotFound
	}
	return nil
}

// File link operations

func (r *fileRepository) CreateFileLink(ctx context.Context, link *model.FileLink) error {
	query := `
		INSERT INTO con_test.file_links (id, file_id, uploaded_by, uploaded_at, is_deleted)
		VALUES ($1, $2, $3, $4, $5)
	`

	link.ID = uuid.New()
	link.UploadedAt = time.Now()
	link.IsDeleted = false

	_, err := r.pool.Exec(ctx, query, link.ID, link.FileID, link.UploadedBy, link.UploadedAt, link.IsDeleted)
	if err != nil {
		return fmt.Errorf("failed to create file link: %w", err)
	}

	return nil
}

func (r *fileRepository) GetFileLink(ctx context.Context, id uuid.UUID) (*model.FileLink, error) {
	query := `
		SELECT id, file_id, uploaded_by, uploaded_at, is_deleted
		FROM con_test.file_links
		WHERE id = $1
	`

	var link model.FileLink
	err := r.pool.QueryRow(ctx, query, id).Scan(&link.ID, &link.FileID, &link.UploadedBy, &link.UploadedAt, &link.IsDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFileLinkNotFound
		}
		return nil, fmt.Errorf("failed to get file link: %w", err)
	}

	return &link, nil
}

func (r *fileRepository) GetFileLinkByFileID(ctx context.Context, fileID uuid.UUID) (*model.FileLink, error) {
	query := `
		SELECT id, file_id, uploaded_by, uploaded_at, is_deleted
		FROM con_test.file_links
		WHERE file_id = $1 AND is_deleted = false
		ORDER BY uploaded_at DESC
		LIMIT 1
	`

	var link model.FileLink
	err := r.pool.QueryRow(ctx, query, fileID).Scan(&link.ID, &link.FileID, &link.UploadedBy, &link.UploadedAt, &link.IsDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFileLinkNotFound
		}
		return nil, fmt.Errorf("failed to get file link: %w", err)
	}

	return &link, nil
}

func (r *fileRepository) DeleteFileLink(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM con_test.file_links WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file link: %w", err)
	}
	return nil
}

func (r *fileRepository) SoftDeleteFileLink(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE con_test.file_links SET is_deleted = true WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete file link: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrFileLinkNotFound
	}
	return nil
}

// Permissions

func (r *fileRepository) CreateFileLinkPermission(ctx context.Context, perm *model.FileLinkPermission) error {
	query := `
		INSERT INTO con_test.file_link_permissions (id, file_link_id, user_id, can_view, can_download, can_delete)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (file_link_id, user_id) DO UPDATE
		SET can_view = $4, can_download = $5, can_delete = $6
	`

	perm.ID = uuid.New()

	_, err := r.pool.Exec(ctx, query, perm.ID, perm.FileLinkID, perm.UserID, perm.CanView, perm.CanDownload, perm.CanDelete)
	if err != nil {
		return fmt.Errorf("failed to create file link permission: %w", err)
	}

	return nil
}

func (r *fileRepository) GetFileLinkPermission(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileLinkPermission, error) {
	query := `
		SELECT id, file_link_id, user_id, can_view, can_download, can_delete
		FROM con_test.file_link_permissions
		WHERE file_link_id = $1 AND user_id = $2
	`

	var perm model.FileLinkPermission
	err := r.pool.QueryRow(ctx, query, fileLinkID, userID).Scan(
		&perm.ID, &perm.FileLinkID, &perm.UserID, &perm.CanView, &perm.CanDownload, &perm.CanDelete,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccessDenied
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &perm, nil
}

func (r *fileRepository) CreatePermissionsForParticipants(ctx context.Context, fileLinkID uuid.UUID, participantIDs []uuid.UUID, uploaderID uuid.UUID) error {
	for _, userID := range participantIDs {
		perm := &model.FileLinkPermission{
			FileLinkID:  fileLinkID,
			UserID:      userID,
			CanView:     true,
			CanDownload: true,
			CanDelete:   userID == uploaderID,
		}
		if err := r.CreateFileLinkPermission(ctx, perm); err != nil {
			return err
		}
	}
	return nil
}

// Share links

func (r *fileRepository) CreateShareLink(ctx context.Context, link *model.FileShareLink) error {
	query := `
		INSERT INTO con_test.file_share_links (id, file_id, token, password, max_downloads, download_count, created_by, created_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	link.ID = uuid.New()
	link.CreatedAt = time.Now()
	link.DownloadCount = 0
	link.IsActive = true

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	link.Token = hex.EncodeToString(tokenBytes)

	_, err := r.pool.Exec(ctx, query,
		link.ID, link.FileID, link.Token, link.Password, link.MaxDownloads,
		link.DownloadCount, link.CreatedBy, link.CreatedAt, link.ExpiresAt, link.IsActive,
	)
	if err != nil {
		return fmt.Errorf("failed to create share link: %w", err)
	}

	return nil
}

func (r *fileRepository) GetShareLinkByToken(ctx context.Context, token string) (*model.FileShareLink, error) {
	query := `
		SELECT id, file_id, token, password, max_downloads, download_count, created_by, created_at, expires_at, is_active
		FROM con_test.file_share_links
		WHERE token = $1
	`

	var link model.FileShareLink
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&link.ID, &link.FileID, &link.Token, &link.Password, &link.MaxDownloads,
		&link.DownloadCount, &link.CreatedBy, &link.CreatedAt, &link.ExpiresAt, &link.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrShareLinkNotFound
		}
		return nil, fmt.Errorf("failed to get share link: %w", err)
	}

	// Check if expired
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, ErrShareLinkExpired
	}

	// Check if active
	if !link.IsActive {
		return nil, ErrShareLinkInactive
	}

	// Check max downloads
	if link.MaxDownloads != nil && link.DownloadCount >= *link.MaxDownloads {
		return nil, ErrMaxDownloads
	}

	return &link, nil
}

func (r *fileRepository) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE con_test.file_share_links SET download_count = download_count + 1 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment download count: %w", err)
	}
	return nil
}

func (r *fileRepository) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE con_test.file_share_links SET is_active = false WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate share link: %w", err)
	}
	return nil
}

// Message attachments

func (r *fileRepository) CreateMessageAttachment(ctx context.Context, attachment *model.MessageFileAttachment) error {
	query := `
		INSERT INTO con_test.message_file_attachments (id, message_id, file_link_id, sort_order)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, file_link_id) DO NOTHING
	`

	attachment.ID = uuid.New()

	_, err := r.pool.Exec(ctx, query, attachment.ID, attachment.MessageID, attachment.FileLinkID, attachment.SortOrder)
	if err != nil {
		return fmt.Errorf("failed to create message attachment: %w", err)
	}

	return nil
}

func (r *fileRepository) GetMessageAttachments(ctx context.Context, messageID uuid.UUID) ([]model.MessageFileAttachment, error) {
	query := `
		SELECT id, message_id, file_link_id, sort_order
		FROM con_test.message_file_attachments
		WHERE message_id = $1
		ORDER BY sort_order
	`

	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message attachments: %w", err)
	}
	defer rows.Close()

	var attachments []model.MessageFileAttachment
	for rows.Next() {
		var a model.MessageFileAttachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.FileLinkID, &a.SortOrder); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, a)
	}

	return attachments, nil
}
