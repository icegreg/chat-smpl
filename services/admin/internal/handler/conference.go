package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/icegreg/chat-smpl/services/admin/internal/model"
	"github.com/icegreg/chat-smpl/services/admin/internal/service"
	"go.uber.org/zap"
)

// ConferenceHandler handles conference HTTP endpoints
type ConferenceHandler struct {
	service *service.ConferenceService
	logger  *zap.Logger
}

// NewConferenceHandler creates a new conference handler
func NewConferenceHandler(service *service.ConferenceService, logger *zap.Logger) *ConferenceHandler {
	return &ConferenceHandler{
		service: service,
		logger:  logger,
	}
}

// ListConferences handles GET /api/conferences
func (h *ConferenceHandler) ListConferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query params
	statusParam := r.URL.Query().Get("status")
	var status *model.ConferenceStatus
	if statusParam != "" {
		s := model.ConferenceStatus(statusParam)
		status = &s
	}

	conferences, err := h.service.ListConferences(ctx, status)
	if err != nil {
		h.logger.Error("failed to list conferences", zap.Error(err))
		h.writeError(w, http.StatusInternalServerError, "failed to list conferences", err)
		return
	}

	resp := model.ConferencesResponse{
		Conferences: conferences,
		Total:       len(conferences),
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// GetConference handles GET /api/conferences/{id}
func (h *ConferenceHandler) GetConference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid conference ID", err)
		return
	}

	conference, err := h.service.GetConference(ctx, id)
	if err != nil {
		h.logger.Error("failed to get conference", zap.Error(err), zap.String("id", id.String()))
		h.writeError(w, http.StatusNotFound, "conference not found", err)
		return
	}

	h.writeJSON(w, http.StatusOK, conference)
}

// ListParticipants handles GET /api/conferences/{id}/participants
func (h *ConferenceHandler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid conference ID", err)
		return
	}

	participants, err := h.service.ListParticipants(ctx, id)
	if err != nil {
		h.logger.Error("failed to list participants", zap.Error(err), zap.String("conference_id", id.String()))
		h.writeError(w, http.StatusInternalServerError, "failed to list participants", err)
		return
	}

	resp := model.ParticipantsResponse{
		Participants: participants,
		Total:        len(participants),
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// EndConference handles POST /api/conferences/{id}/end
func (h *ConferenceHandler) EndConference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid conference ID", err)
		return
	}

	// Parse request body for user_id
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid user_id UUID", err)
		return
	}

	if err := h.service.EndConference(ctx, id, userID); err != nil {
		h.logger.Error("failed to end conference", zap.Error(err), zap.String("id", id.String()))
		h.writeError(w, http.StatusInternalServerError, "failed to end conference", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"status": "conference ended"})
}

// writeJSON writes JSON response
func (h *ConferenceHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode JSON response", zap.Error(err))
	}
}

// writeError writes error response
func (h *ConferenceHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	resp := model.ErrorResponse{
		Error:   message,
		Message: errMsg,
	}

	h.writeJSON(w, status, resp)
}
