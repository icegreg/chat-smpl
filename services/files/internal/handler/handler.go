package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/files/internal/model"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
	"github.com/icegreg/chat-smpl/services/files/internal/service"
)

type Handler struct {
	fileService service.FileService
	log         logger.Logger
	maxFileSize int64
}

func NewHandler(fileService service.FileService, log logger.Logger, maxFileSize int64) *Handler {
	return &Handler{
		fileService: fileService,
		log:         log,
		maxFileSize: maxFileSize,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// File operations (requires authentication via middleware)
	r.Post("/upload", h.Upload)
	r.Get("/{linkId}", h.Download)
	r.Get("/{linkId}/info", h.GetFileInfo)
	r.Delete("/{linkId}", h.Delete)

	// Share links
	r.Post("/{fileId}/share", h.CreateShareLink)
	r.Get("/share/{token}", h.DownloadByShareToken)

	return r
}

// Upload handles file upload
// POST /files/upload
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxFileSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(h.maxFileSize); err != nil {
		h.log.Error("failed to parse multipart form", "error", err)
		h.respondError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		h.log.Error("failed to get file from form", "error", err)
		h.respondError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	// Get content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload file
	response, err := h.fileService.Upload(ctx, header.Filename, contentType, header.Size, file, userID)
	if err != nil {
		h.log.Error("failed to upload file", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	h.respondJSON(w, http.StatusCreated, response)
}

// Download handles file download
// GET /files/{linkId}
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	linkIDStr := chi.URLParam(r, "linkId")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid link ID")
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reader, file, err := h.fileService.Download(ctx, linkID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	defer reader.Close()

	// Set response headers
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalFilename))
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

	// Stream file to response
	if _, err := io.Copy(w, reader); err != nil {
		h.log.Error("failed to stream file", "error", err)
	}
}

// GetFileInfo returns file metadata
// GET /files/{linkId}/info
func (h *Handler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	linkIDStr := chi.URLParam(r, "linkId")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid link ID")
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileDTO, err := h.fileService.GetFileInfo(ctx, linkID, userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, fileDTO)
}

// Delete handles file deletion
// DELETE /files/{linkId}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	linkIDStr := chi.URLParam(r, "linkId")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid link ID")
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.fileService.Delete(ctx, linkID, userID); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateShareLink creates a public share link for a file
// POST /files/{fileId}/share
func (h *Handler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	fileIDStr := chi.URLParam(r, "fileId")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid file ID")
		return
	}

	userID, err := getUserIDFromContext(r)
	if err != nil {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req model.CreateShareLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	shareLink, err := h.fileService.CreateShareLink(ctx, fileID, userID, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, shareLink)
}

// DownloadByShareToken downloads file using share token
// GET /files/share/{token}
func (h *Handler) DownloadByShareToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := chi.URLParam(r, "token")
	if token == "" {
		h.respondError(w, http.StatusBadRequest, "token required")
		return
	}

	// Get password from query or header
	password := r.URL.Query().Get("password")
	if password == "" {
		password = r.Header.Get("X-Share-Password")
	}

	reader, file, err := h.fileService.DownloadByShareToken(ctx, token, password)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	defer reader.Close()

	// Set response headers
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.OriginalFilename))
	w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

	// Stream file to response
	if _, err := io.Copy(w, reader); err != nil {
		h.log.Error("failed to stream file", "error", err)
	}
}

// Helper functions

func getUserIDFromContext(r *http.Request) (uuid.UUID, error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}
	return uuid.Parse(userIDStr)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			h.log.Error("failed to encode response", "error", err)
		}
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch err {
	case repository.ErrFileNotFound, repository.ErrFileLinkNotFound:
		h.respondError(w, http.StatusNotFound, "file not found")
	case repository.ErrAccessDenied:
		h.respondError(w, http.StatusForbidden, "access denied")
	case repository.ErrShareLinkNotFound:
		h.respondError(w, http.StatusNotFound, "share link not found")
	case repository.ErrShareLinkExpired:
		h.respondError(w, http.StatusGone, "share link expired")
	case repository.ErrShareLinkInactive:
		h.respondError(w, http.StatusGone, "share link is no longer active")
	case repository.ErrMaxDownloads:
		h.respondError(w, http.StatusGone, "maximum downloads reached")
	default:
		h.log.Error("service error", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal server error")
	}
}
