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
	r.Get("/{linkId}", h.Download)

	return r
}

// Upload proxies file upload to files service
// POST /api/files/upload
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

// Download proxies file download from files service
// GET /api/files/{linkId}
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

func (h *FilesHandler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
