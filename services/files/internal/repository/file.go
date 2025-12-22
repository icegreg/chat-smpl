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

	// Individual permissions
	CreateFileLinkPermission(ctx context.Context, perm *model.FileLinkPermission) error
	GetFileLinkPermission(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileLinkPermission, error)
	CheckFileAccess(ctx context.Context, fileLinkID, userID uuid.UUID) (model.FileAccessLevel, error)
	DeletePermissionsForUser(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error

	// File groups (Files Service owns groups)
	CreateFileGroup(ctx context.Context, group *model.FileGroup) error
	GetFileGroup(ctx context.Context, id uuid.UUID) (*model.FileGroup, error)
	DeleteFileGroup(ctx context.Context, id uuid.UUID) error

	// Group membership
	AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error
	RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	GetUserGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)

	// File-group links
	AddFileLinkToGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error
	RemoveFileLinkFromGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error
	GetFileLinkGroups(ctx context.Context, fileLinkID uuid.UUID) ([]uuid.UUID, error)
	GetFilesByGroup(ctx context.Context, groupID uuid.UUID) ([]model.FileLink, error)

	// Share links
	CreateShareLink(ctx context.Context, link *model.FileShareLink) error
	GetShareLinkByToken(ctx context.Context, token string) (*model.FileShareLink, error)
	IncrementDownloadCount(ctx context.Context, id uuid.UUID) error
	DeactivateShareLink(ctx context.Context, id uuid.UUID) error

	// Message attachments
	CreateMessageAttachment(ctx context.Context, attachment *model.MessageFileAttachment) error
	GetMessageAttachments(ctx context.Context, messageID uuid.UUID) ([]model.MessageFileAttachment, error)

	// Batch operations
	GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) (map[uuid.UUID]*model.File, error)
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

func (r *fileRepository) DeletePermissionsForUser(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error {
	if len(linkIDs) == 0 {
		return nil
	}

	query := `
		DELETE FROM con_test.file_link_permissions
		WHERE file_link_id = ANY($1) AND user_id = $2
	`

	_, err := r.pool.Exec(ctx, query, linkIDs, userID)
	if err != nil {
		return fmt.Errorf("failed to delete permissions: %w", err)
	}
	return nil
}

// CheckFileAccess uses the PostgreSQL function to check user's access level to a file
func (r *fileRepository) CheckFileAccess(ctx context.Context, fileLinkID, userID uuid.UUID) (model.FileAccessLevel, error) {
	query := `SELECT con_test.check_file_access($1, $2)`

	var accessLevel string
	err := r.pool.QueryRow(ctx, query, fileLinkID, userID).Scan(&accessLevel)
	if err != nil {
		return model.FileAccessNone, fmt.Errorf("failed to check file access: %w", err)
	}

	return model.FileAccessLevel(accessLevel), nil
}

// File groups

func (r *fileRepository) CreateFileGroup(ctx context.Context, group *model.FileGroup) error {
	query := `
		INSERT INTO con_test.file_groups (id, name, can_read, can_delete, can_transfer, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	group.ID = uuid.New()
	group.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query, group.ID, group.Name, group.CanRead, group.CanDelete, group.CanTransfer, group.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create file group: %w", err)
	}
	return nil
}

func (r *fileRepository) GetFileGroup(ctx context.Context, id uuid.UUID) (*model.FileGroup, error) {
	query := `
		SELECT id, name, can_read, can_delete, can_transfer, created_at
		FROM con_test.file_groups
		WHERE id = $1
	`

	var group model.FileGroup
	err := r.pool.QueryRow(ctx, query, id).Scan(&group.ID, &group.Name, &group.CanRead, &group.CanDelete, &group.CanTransfer, &group.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("file group not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get file group: %w", err)
	}
	return &group, nil
}

func (r *fileRepository) DeleteFileGroup(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM con_test.file_groups WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete file group: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("file group not found: %s", id)
	}
	return nil
}

// Group membership

func (r *fileRepository) AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	query := `
		INSERT INTO con_test.file_group_members (group_id, user_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (group_id, user_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, groupID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}
	return nil
}

func (r *fileRepository) RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	query := `DELETE FROM con_test.file_group_members WHERE group_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from group: %w", err)
	}
	return nil
}

func (r *fileRepository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT user_id FROM con_test.file_group_members WHERE group_id = $1`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var members []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan user ID: %w", err)
		}
		members = append(members, userID)
	}
	return members, nil
}

func (r *fileRepository) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT group_id FROM con_test.file_group_members WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user groups: %w", err)
	}
	defer rows.Close()

	var groups []uuid.UUID
	for rows.Next() {
		var groupID uuid.UUID
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("failed to scan group ID: %w", err)
		}
		groups = append(groups, groupID)
	}
	return groups, nil
}

// File-group links

func (r *fileRepository) AddFileLinkToGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error {
	query := `
		INSERT INTO con_test.file_link_groups (file_link_id, group_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (file_link_id, group_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query, fileLinkID, groupID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add file link to group: %w", err)
	}
	return nil
}

func (r *fileRepository) RemoveFileLinkFromGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error {
	query := `DELETE FROM con_test.file_link_groups WHERE file_link_id = $1 AND group_id = $2`
	_, err := r.pool.Exec(ctx, query, fileLinkID, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove file link from group: %w", err)
	}
	return nil
}

func (r *fileRepository) GetFileLinkGroups(ctx context.Context, fileLinkID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT group_id FROM con_test.file_link_groups WHERE file_link_id = $1`

	rows, err := r.pool.Query(ctx, query, fileLinkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file link groups: %w", err)
	}
	defer rows.Close()

	var groups []uuid.UUID
	for rows.Next() {
		var groupID uuid.UUID
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("failed to scan group ID: %w", err)
		}
		groups = append(groups, groupID)
	}
	return groups, nil
}

func (r *fileRepository) GetFilesByGroup(ctx context.Context, groupID uuid.UUID) ([]model.FileLink, error) {
	query := `
		SELECT fl.id, fl.file_id, fl.uploaded_by, fl.uploaded_at, fl.is_deleted
		FROM con_test.file_link_groups flg
		JOIN con_test.file_links fl ON fl.id = flg.file_link_id
		WHERE flg.group_id = $1 AND fl.is_deleted = FALSE
	`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get files by group: %w", err)
	}
	defer rows.Close()

	var links []model.FileLink
	for rows.Next() {
		var link model.FileLink
		if err := rows.Scan(&link.ID, &link.FileID, &link.UploadedBy, &link.UploadedAt, &link.IsDeleted); err != nil {
			return nil, fmt.Errorf("failed to scan file link: %w", err)
		}
		links = append(links, link)
	}
	return links, nil
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

func (r *fileRepository) GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) (map[uuid.UUID]*model.File, error) {
	if len(linkIDs) == 0 {
		return make(map[uuid.UUID]*model.File), nil
	}

	query := `
		SELECT fl.id as link_id, f.id, f.filename, f.original_filename, f.content_type, f.size, f.uploaded_by, f.uploaded_at
		FROM con_test.file_links fl
		JOIN con_test.files f ON fl.file_id = f.id
		WHERE fl.id = ANY($1) AND fl.is_deleted = false AND f.status = 'active'
	`

	rows, err := r.pool.Query(ctx, query, linkIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get files by link IDs: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]*model.File)
	for rows.Next() {
		var linkID uuid.UUID
		var f model.File
		if err := rows.Scan(&linkID, &f.ID, &f.Filename, &f.OriginalFilename, &f.ContentType, &f.Size, &f.UploadedBy, &f.UploadedAt); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		result[linkID] = &f
	}

	return result, nil
}
