package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

type FilesHandler struct {
	filesServiceURL string
	log             logger.Logger
}

func NewFilesHandler(filesServiceURL string, log logger.Logger) *FilesHandler {
	return &FilesHandler{
		filesServiceURL: strings.TrimSuffix(filesServiceURL, "/"),
		log:             log,
	}
}

func (h *FilesHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/upload", h.Upload)
	r.Get("/chats/{chatId}/files", h.GetChatFiles)
	r.Get("/{linkId}", h.Download)
	r.Get("/{linkId}/info", h.GetFileInfo)
	r.Delete("/{linkId}", h.Delete)

	return r
}

// Upload godoc
// @Summary Upload a file
// @Description Uploads a file to the storage service
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param file formData file true "File to upload"
// @Success 201 {object} FileUploadResponse "File uploaded successfully"
// @Failure 400 {object} ErrorResponse "Invalid file"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 502 {object} ErrorResponse "Files service unavailable"
// @Router /files/upload [post]
func (h *FilesHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Create proxy request
	proxyURL := h.filesServiceURL + "/files/upload"
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodPost, proxyURL, r.Body)
	if err != nil {
		h.log.Error("failed to create proxy request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Copy content-type header for multipart form
	proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	proxyReq.Header.Set("X-User-ID", userID.String())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		h.log.Error("failed to proxy upload request", "error", err)
		h.respondError(w, http.StatusBadGateway, "files service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Download godoc
// @Summary Download a file
// @Description Downloads a file by its link ID
// @Tags files
// @Produce octet-stream
// @Security Bearer
// @Param linkId path string true "File link ID"
// @Success 200 {file} binary "File content"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "File not found"
// @Failure 502 {object} ErrorResponse "Files service unavailable"
// @Router /files/{linkId} [get]
func (h *FilesHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	linkId := chi.URLParam(r, "linkId")

	userID, _ := middleware.GetUserID(ctx)

	// Create proxy request
	proxyURL := h.filesServiceURL + "/files/" + linkId
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL, nil)
	if err != nil {
		h.log.Error("failed to create proxy request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if userID != uuid.Nil {
		proxyReq.Header.Set("X-User-ID", userID.String())
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		h.log.Error("failed to proxy download request", "error", err)
		h.respondError(w, http.StatusBadGateway, "files service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GetFileInfo godoc
// @Summary Get file information
// @Description Returns file metadata by its link ID
// @Tags files
// @Produce json
// @Security Bearer
// @Param linkId path string true "File link ID"
// @Success 200 {object} map[string]interface{} "File information"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "File not found"
// @Failure 502 {object} ErrorResponse "Files service unavailable"
// @Router /files/{linkId}/info [get]
func (h *FilesHandler) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	linkId := chi.URLParam(r, "linkId")

	userID, _ := middleware.GetUserID(ctx)

	// Create proxy request
	proxyURL := h.filesServiceURL + "/files/" + linkId + "/info"
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL, nil)
	if err != nil {
		h.log.Error("failed to create proxy request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if userID != uuid.Nil {
		proxyReq.Header.Set("X-User-ID", userID.String())
	}

	// Send request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		h.log.Error("failed to proxy file info request", "error", err)
		h.respondError(w, http.StatusBadGateway, "files service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Delete godoc
// @Summary Delete a file
// @Description Deletes a file by its link ID
// @Tags files
// @Security Bearer
// @Param linkId path string true "File link ID"
// @Success 204 "File deleted"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "File not found"
// @Failure 502 {object} ErrorResponse "Files service unavailable"
// @Router /files/{linkId} [delete]
func (h *FilesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	linkId := chi.URLParam(r, "linkId")

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Create proxy request
	proxyURL := h.filesServiceURL + "/files/" + linkId
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, proxyURL, nil)
	if err != nil {
		h.log.Error("failed to create proxy request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	proxyReq.Header.Set("X-User-ID", userID.String())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		h.log.Error("failed to proxy delete request", "error", err)
		h.respondError(w, http.StatusBadGateway, "files service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent {
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		io.Copy(w, resp.Body)
	}
}

// GetChatFiles godoc
// @Summary Get files in a chat
// @Description Returns all files uploaded to a specific chat
// @Tags files
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param limit query int false "Number of files to return" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} ChatFilesResponse "List of files"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 502 {object} ErrorResponse "Files service unavailable"
// @Router /files/chats/{chatId}/files [get]
func (h *FilesHandler) GetChatFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatId := chi.URLParam(r, "chatId")

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get query params
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	if limit == "" {
		limit = "50"
	}
	if offset == "" {
		offset = "0"
	}

	// Create proxy request
	proxyURL := h.filesServiceURL + "/files/chats/" + chatId + "/files?limit=" + limit + "&offset=" + offset
	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL, nil)
	if err != nil {
		h.log.Error("failed to create proxy request", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	proxyReq.Header.Set("X-User-ID", userID.String())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		h.log.Error("failed to proxy get chat files request", "error", err)
		h.respondError(w, http.StatusBadGateway, "files service unavailable")
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (h *FilesHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
