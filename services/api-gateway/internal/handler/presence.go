package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"

	pb "github.com/icegreg/chat-smpl/proto/presence"
)

type PresenceHandler struct {
	presenceClient *grpc.PresenceClient
	log            logger.Logger
}

func NewPresenceHandler(presenceClient *grpc.PresenceClient, log logger.Logger) *PresenceHandler {
	return &PresenceHandler{
		presenceClient: presenceClient,
		log:            log,
	}
}

func (h *PresenceHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Put("/status", h.SetStatus)
	r.Get("/status", h.GetMyStatus)
	r.Get("/users", h.GetUsersPresence)
	r.Post("/connect", h.Connect)
	r.Post("/disconnect", h.Disconnect)

	return r
}

// SetStatusRequest represents the request body for setting status
type SetStatusRequest struct {
	Status string `json:"status"` // available, busy, away, dnd
}

// PresenceResponse represents the response for presence info
type PresenceResponse struct {
	UserID          string `json:"user_id"`
	Status          string `json:"status"`
	IsOnline        bool   `json:"is_online"`
	ConnectionCount int32  `json:"connection_count"`
	LastSeenAt      int64  `json:"last_seen_at,omitempty"`
}

// SetStatus sets the current user's status
// PUT /api/presence/status
func (h *PresenceHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req SetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert string status to proto enum
	status := stringToStatus(req.Status)
	if status == pb.UserStatus_STATUS_UNSPECIFIED {
		h.writeError(w, http.StatusBadRequest, "invalid status, must be one of: available, busy, away, dnd")
		return
	}

	presence, err := h.presenceClient.SetStatus(r.Context(), userID.String(), status)
	if err != nil {
		h.log.Error("failed to set status", "error", err, "user_id", userID)
		h.writeError(w, http.StatusInternalServerError, "failed to set status")
		return
	}

	h.writeJSON(w, http.StatusOK, presenceToResponse(presence))
}

// GetMyStatus gets the current user's presence info
// GET /api/presence/status
func (h *PresenceHandler) GetMyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	presence, err := h.presenceClient.GetPresence(r.Context(), userID.String())
	if err != nil {
		h.log.Error("failed to get presence", "error", err, "user_id", userID)
		h.writeError(w, http.StatusInternalServerError, "failed to get presence")
		return
	}

	h.writeJSON(w, http.StatusOK, presenceToResponse(presence))
}

// GetUsersPresence gets presence info for multiple users
// GET /api/presence/users?ids=user1,user2,user3
func (h *PresenceHandler) GetUsersPresence(w http.ResponseWriter, r *http.Request) {
	idsParam := r.URL.Query().Get("ids")
	if idsParam == "" {
		h.writeError(w, http.StatusBadRequest, "ids parameter is required")
		return
	}

	// Parse comma-separated user IDs
	var userIDs []string
	for _, id := range splitIDs(idsParam) {
		if id != "" {
			userIDs = append(userIDs, id)
		}
	}

	if len(userIDs) == 0 {
		h.writeError(w, http.StatusBadRequest, "at least one user id is required")
		return
	}

	if len(userIDs) > 100 {
		h.writeError(w, http.StatusBadRequest, "max 100 user ids allowed")
		return
	}

	presences, err := h.presenceClient.GetPresencesBatch(r.Context(), userIDs)
	if err != nil {
		h.log.Error("failed to get presences batch", "error", err, "user_ids", userIDs)
		h.writeError(w, http.StatusInternalServerError, "failed to get presences")
		return
	}

	response := make([]PresenceResponse, 0, len(presences))
	for _, p := range presences {
		response = append(response, presenceToResponse(p))
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"presences": response,
	})
}

// Helper functions

func stringToStatus(s string) pb.UserStatus {
	switch s {
	case "available":
		return pb.UserStatus_STATUS_AVAILABLE
	case "busy":
		return pb.UserStatus_STATUS_BUSY
	case "away":
		return pb.UserStatus_STATUS_AWAY
	case "dnd":
		return pb.UserStatus_STATUS_DND
	default:
		return pb.UserStatus_STATUS_UNSPECIFIED
	}
}

func statusToString(s pb.UserStatus) string {
	switch s {
	case pb.UserStatus_STATUS_AVAILABLE:
		return "available"
	case pb.UserStatus_STATUS_BUSY:
		return "busy"
	case pb.UserStatus_STATUS_AWAY:
		return "away"
	case pb.UserStatus_STATUS_DND:
		return "dnd"
	default:
		return "unknown"
	}
}

func presenceToResponse(p *pb.PresenceInfo) PresenceResponse {
	var lastSeenAt int64
	if p.LastSeenAt != "" {
		lastSeenAt, _ = strconv.ParseInt(p.LastSeenAt, 10, 64)
	}
	return PresenceResponse{
		UserID:          p.UserId,
		Status:          statusToString(p.Status),
		IsOnline:        p.IsOnline,
		ConnectionCount: p.ConnectionCount,
		LastSeenAt:      lastSeenAt,
	}
}

func splitIDs(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func (h *PresenceHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *PresenceHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// ConnectRequest represents the request body for connect
type ConnectRequest struct {
	ConnectionID string `json:"connection_id"`
}

// Connect registers a new WebSocket connection for the user
// POST /api/presence/connect
func (h *PresenceHandler) Connect(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ConnectionID == "" {
		h.writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}

	presence, err := h.presenceClient.UserConnected(r.Context(), userID.String(), req.ConnectionID)
	if err != nil {
		h.log.Error("failed to register connection", "error", err, "user_id", userID)
		h.writeError(w, http.StatusInternalServerError, "failed to register connection")
		return
	}

	h.writeJSON(w, http.StatusOK, presenceToResponse(presence))
}

// Disconnect unregisters a WebSocket connection for the user
// POST /api/presence/disconnect
func (h *PresenceHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ConnectionID == "" {
		h.writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}

	presence, err := h.presenceClient.UserDisconnected(r.Context(), userID.String(), req.ConnectionID)
	if err != nil {
		h.log.Error("failed to unregister connection", "error", err, "user_id", userID)
		h.writeError(w, http.StatusInternalServerError, "failed to unregister connection")
		return
	}

	h.writeJSON(w, http.StatusOK, presenceToResponse(presence))
}
