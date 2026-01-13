package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type FileStatus string

const (
	FileStatusActive  FileStatus = "active"
	FileStatusDeleted FileStatus = "deleted"
)

type File struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	Filename         string          `json:"filename" db:"filename"`
	OriginalFilename string          `json:"original_filename" db:"original_filename"`
	ContentType      string          `json:"content_type" db:"content_type"`
	Size             int64           `json:"size" db:"size"`
	FilePath         string          `json:"-" db:"file_path"`
	UploadedBy       uuid.UUID       `json:"uploaded_by" db:"uploaded_by"`
	UploadedAt       time.Time       `json:"uploaded_at" db:"uploaded_at"`
	Status           FileStatus      `json:"status" db:"status"`
	Metadata         json.RawMessage `json:"metadata,omitempty" db:"metadata"`
}

type FileLink struct {
	ID         uuid.UUID `json:"id" db:"id"`
	FileID     uuid.UUID `json:"file_id" db:"file_id"`
	UploadedBy uuid.UUID `json:"uploaded_by" db:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
	IsDeleted  bool      `json:"is_deleted" db:"is_deleted"`
}

// FileAccessLevel represents the level of access a user has to a file
type FileAccessLevel string

const (
	FileAccessNone     FileAccessLevel = "none"
	FileAccessRead     FileAccessLevel = "read"     // Can view/download
	FileAccessDelete   FileAccessLevel = "delete"   // Can read + delete
	FileAccessTransfer FileAccessLevel = "transfer" // Can read + delete + transfer ownership
)

// FileGroup represents a group with specific permissions
type FileGroup struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	CanRead     bool      `json:"can_read" db:"can_read"`
	CanDelete   bool      `json:"can_delete" db:"can_delete"`
	CanTransfer bool      `json:"can_transfer" db:"can_transfer"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// FileGroupMember represents membership in a file group
type FileGroupMember struct {
	GroupID uuid.UUID `json:"group_id" db:"group_id"`
	UserID  uuid.UUID `json:"user_id" db:"user_id"`
	AddedAt time.Time `json:"added_at" db:"added_at"`
}

// FileLinkGroup links a file_link to a group
type FileLinkGroup struct {
	FileLinkID uuid.UUID `json:"file_link_id" db:"file_link_id"`
	GroupID    uuid.UUID `json:"group_id" db:"group_id"`
	AddedAt    time.Time `json:"added_at" db:"added_at"`
}

type FileLinkPermission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	FileLinkID  uuid.UUID `json:"file_link_id" db:"file_link_id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	CanView     bool      `json:"can_view" db:"can_view"`
	CanDownload bool      `json:"can_download" db:"can_download"`
	CanDelete   bool      `json:"can_delete" db:"can_delete"`
}

type FileShareLink struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	FileID        uuid.UUID  `json:"file_id" db:"file_id"`
	Token         string     `json:"token" db:"token"`
	Password      *string    `json:"-" db:"password"`
	MaxDownloads  *int       `json:"max_downloads,omitempty" db:"max_downloads"`
	DownloadCount int        `json:"download_count" db:"download_count"`
	CreatedBy     uuid.UUID  `json:"created_by" db:"created_by"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsActive      bool       `json:"is_active" db:"is_active"`
}

type MessageFileAttachment struct {
	ID         uuid.UUID `json:"id" db:"id"`
	MessageID  uuid.UUID `json:"message_id" db:"message_id"`
	FileLinkID uuid.UUID `json:"file_link_id" db:"file_link_id"`
	SortOrder  int       `json:"sort_order" db:"sort_order"`
}

// DTOs

type UploadFileResponse struct {
	ID               uuid.UUID `json:"id"`
	LinkID           uuid.UUID `json:"link_id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	Size             int64     `json:"size"`
}

type FileDTO struct {
	ID               uuid.UUID `json:"id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	Size             int64     `json:"size"`
	UploadedBy       uuid.UUID `json:"uploaded_by"`
	UploadedAt       time.Time `json:"uploaded_at"`
}

func (f *File) ToDTO() FileDTO {
	return FileDTO{
		ID:               f.ID,
		Filename:         f.Filename,
		OriginalFilename: f.OriginalFilename,
		ContentType:      f.ContentType,
		Size:             f.Size,
		UploadedBy:       f.UploadedBy,
		UploadedAt:       f.UploadedAt,
	}
}

type CreateShareLinkRequest struct {
	Password     *string    `json:"password,omitempty"`
	MaxDownloads *int       `json:"max_downloads,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

// FileAttachmentDTO is used for returning file metadata for message attachments
type FileAttachmentDTO struct {
	LinkID           uuid.UUID `json:"link_id"`
	ID               uuid.UUID `json:"id"`
	Filename         string    `json:"filename"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	Size             int64     `json:"size"`
}

type ShareLinkDTO struct {
	ID            uuid.UUID  `json:"id"`
	Token         string     `json:"token"`
	URL           string     `json:"url"`
	MaxDownloads  *int       `json:"max_downloads,omitempty"`
	DownloadCount int        `json:"download_count"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ChatFileDTO represents a file uploaded to a chat
type ChatFileDTO struct {
	LinkID           uuid.UUID `json:"link_id"`
	FileID           uuid.UUID `json:"file_id"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	Size             int64     `json:"size"`
	UploadedBy       uuid.UUID `json:"uploaded_by"`
	UploadedAt       time.Time `json:"uploaded_at"`
	UploaderUsername *string   `json:"uploader_username,omitempty"`
}

func (s *FileShareLink) ToDTO(baseURL string) ShareLinkDTO {
	return ShareLinkDTO{
		ID:            s.ID,
		Token:         s.Token,
		URL:           baseURL + "/share/" + s.Token,
		MaxDownloads:  s.MaxDownloads,
		DownloadCount: s.DownloadCount,
		ExpiresAt:     s.ExpiresAt,
		IsActive:      s.IsActive,
		CreatedAt:     s.CreatedAt,
	}
}
