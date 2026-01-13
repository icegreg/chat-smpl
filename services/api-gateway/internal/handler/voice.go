package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
	pb "github.com/icegreg/chat-smpl/proto/voice"
)

type VoiceHandler struct {
	voiceClient *grpc.VoiceClient
	log         logger.Logger
}

func NewVoiceHandler(voiceClient *grpc.VoiceClient, log logger.Logger) *VoiceHandler {
	return &VoiceHandler{
		voiceClient: voiceClient,
		log:         log,
	}
}

func (h *VoiceHandler) Routes() http.Handler {
	r := chi.NewRouter()

	// Conferences
	r.Post("/conferences", h.CreateConference)
	r.Get("/conferences", h.ListConferences)
	r.Get("/conferences/active", h.ListAllActiveConferences)
	r.Get("/conferences/by-fs-name/{fsName}", h.GetConferenceByFSName)
	r.Get("/conferences/{conferenceID}", h.GetConference)
	r.Post("/conferences/{conferenceID}/join", h.JoinConference)
	r.Post("/conferences/{conferenceID}/leave", h.LeaveConference)
	r.Get("/conferences/{conferenceID}/participants", h.GetParticipants)
	r.Post("/conferences/{conferenceID}/participants/{userID}/mute", h.MuteParticipant)
	r.Post("/conferences/{conferenceID}/participants/{userID}/kick", h.KickParticipant)
	r.Post("/conferences/{conferenceID}/end", h.EndConference)

	// Scheduled events
	r.Post("/conferences/schedule", h.ScheduleConference)
	r.Get("/conferences/scheduled", h.ListScheduledConferences)
	r.Post("/conferences/adhoc-chat", h.CreateAdHocFromChat)
	r.Post("/conferences/quick-adhoc", h.CreateQuickAdHoc)
	r.Put("/conferences/{conferenceID}/rsvp", h.UpdateRSVP)
	r.Put("/conferences/{conferenceID}/participants/{userID}/role", h.UpdateParticipantRole)
	r.Post("/conferences/{conferenceID}/participants", h.AddParticipants)
	r.Delete("/conferences/{conferenceID}/participants/{userID}", h.RemoveParticipant)
	r.Delete("/conferences/{conferenceID}", h.CancelConference)

	// Chat conferences
	r.Get("/chats/{chatID}/conferences", h.GetChatConferences)
	r.Get("/chats/{chatID}/conferences/history", h.ListChatConferenceHistory)

	// Conference history
	r.Get("/conferences/{conferenceID}/history", h.GetConferenceHistory)
	r.Get("/conferences/{conferenceID}/messages", h.GetConferenceMessages)
	r.Get("/conferences/{conferenceID}/moderator-actions", h.GetModeratorActions)

	// Calls
	r.Post("/calls", h.InitiateCall)
	r.Post("/calls/{callID}/answer", h.AnswerCall)
	r.Post("/calls/{callID}/hangup", h.HangupCall)
	r.Get("/calls/history", h.GetCallHistory)

	// Verto credentials
	r.Get("/credentials", h.GetVertoCredentials)

	// Quick call from chat
	r.Post("/chats/{chatID}/call", h.StartChatCall)

	return r
}

// CreateConference godoc
// @Summary Create a new conference
// @Description Creates a new voice conference room
// @Tags voice
// @Accept json
// @Produce json
// @Param request body CreateConferenceRequest true "Conference creation request"
// @Success 201 {object} ConferenceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences [post]
func (h *VoiceHandler) CreateConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateConferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	conf, err := h.voiceClient.CreateConference(r.Context(), &pb.CreateConferenceRequest{
		Name:            req.Name,
		ChatId:          req.ChatID,
		CreatedBy:       userID.String(),
		MaxMembers:      int32(req.MaxMembers),
		IsPrivate:       req.IsPrivate,
		EnableRecording: req.EnableRecording,
	})
	if err != nil {
		h.log.Error("failed to create conference", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create conference")
		return
	}

	h.writeJSON(w, http.StatusCreated, conferenceToResponse(conf))
}

// GetConference godoc
// @Summary Get conference details
// @Description Retrieves details of a specific conference
// @Tags voice
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Success 200 {object} ConferenceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID} [get]
func (h *VoiceHandler) GetConference(w http.ResponseWriter, r *http.Request) {
	conferenceID := chi.URLParam(r, "conferenceID")

	conf, err := h.voiceClient.GetConference(r.Context(), conferenceID)
	if err != nil {
		h.log.Error("failed to get conference", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusNotFound, "conference not found")
		return
	}

	h.writeJSON(w, http.StatusOK, conferenceToResponse(conf))
}

// GetConferenceByFSName godoc
// @Summary Get conference by FreeSWITCH name
// @Description Retrieves a conference by its FreeSWITCH conference name
// @Tags voice
// @Produce json
// @Param fsName path string true "FreeSWITCH conference name"
// @Success 200 {object} ConferenceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/by-fs-name/{fsName} [get]
func (h *VoiceHandler) GetConferenceByFSName(w http.ResponseWriter, r *http.Request) {
	fsName := chi.URLParam(r, "fsName")

	conf, err := h.voiceClient.GetConferenceByFSName(r.Context(), fsName)
	if err != nil {
		h.log.Error("failed to get conference by FS name", "error", err, "fsName", fsName)
		h.writeError(w, http.StatusNotFound, "conference not found")
		return
	}

	h.writeJSON(w, http.StatusOK, conferenceToResponse(conf))
}

// ListConferences godoc
// @Summary List conferences
// @Description Lists conferences for the current user
// @Tags voice
// @Produce json
// @Param active_only query bool false "Only show active conferences"
// @Param limit query int false "Number of results to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} ListConferencesResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences [get]
func (h *VoiceHandler) ListConferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	resp, err := h.voiceClient.ListConferences(r.Context(), userID.String(), activeOnly, int32(limit), int32(offset))
	if err != nil {
		// If voice service is unavailable, return empty list
		h.log.Warn("voice service unavailable, returning empty conferences", "error", err)
		h.writeJSON(w, http.StatusOK, ListConferencesResponse{
			Conferences: []ConferenceResponse{},
			Total:       0,
		})
		return
	}

	conferences := make([]ConferenceResponse, len(resp.Conferences))
	for i, conf := range resp.Conferences {
		conferences[i] = conferenceToResponse(conf)
	}

	h.writeJSON(w, http.StatusOK, ListConferencesResponse{
		Conferences: conferences,
		Total:       int(resp.Total),
	})
}

// ListAllActiveConferences godoc
// @Summary List all active conferences with chat_id
// @Description Returns all active conferences that have a chat_id (for UI indicators)
// @Tags voice
// @Produce json
// @Success 200 {object} ListConferencesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/active [get]
func (h *VoiceHandler) ListAllActiveConferences(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.voiceClient.ListAllActiveConferences(r.Context())
	if err != nil {
		// If voice service is unavailable, return empty list
		h.log.Warn("voice service unavailable, returning empty active conferences", "error", err)
		h.writeJSON(w, http.StatusOK, ListConferencesResponse{
			Conferences: []ConferenceResponse{},
			Total:       0,
		})
		return
	}

	conferences := make([]ConferenceResponse, len(resp.Conferences))
	for i, conf := range resp.Conferences {
		conferences[i] = conferenceToResponse(conf)
	}

	h.writeJSON(w, http.StatusOK, ListConferencesResponse{
		Conferences: conferences,
		Total:       int(resp.Total),
	})
}

// JoinConference godoc
// @Summary Join a conference
// @Description Joins the current user to a conference
// @Tags voice
// @Accept json
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Param request body JoinConferenceRequest true "Join options"
// @Success 200 {object} ParticipantResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/join [post]
func (h *VoiceHandler) JoinConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	var req JoinConferenceRequest
	json.NewDecoder(r.Body).Decode(&req) // Ignore errors - optional body

	participant, err := h.voiceClient.JoinConference(r.Context(), conferenceID, userID.String(), req.Muted, req.DisplayName)
	if err != nil {
		h.log.Error("failed to join conference", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusInternalServerError, "failed to join conference")
		return
	}

	h.writeJSON(w, http.StatusOK, participantToResponse(participant))
}

// LeaveConference godoc
// @Summary Leave a conference
// @Description Removes the current user from a conference
// @Tags voice
// @Param conferenceID path string true "Conference ID"
// @Success 204
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/leave [post]
func (h *VoiceHandler) LeaveConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	if err := h.voiceClient.LeaveConference(r.Context(), conferenceID, userID.String()); err != nil {
		h.log.Error("failed to leave conference", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusInternalServerError, "failed to leave conference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetParticipants godoc
// @Summary Get conference participants
// @Description Returns all participants in a conference
// @Tags voice
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Success 200 {object} ParticipantsResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants [get]
func (h *VoiceHandler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	conferenceID := chi.URLParam(r, "conferenceID")

	resp, err := h.voiceClient.GetParticipants(r.Context(), conferenceID)
	if err != nil {
		h.log.Error("failed to get participants", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusInternalServerError, "failed to get participants")
		return
	}

	participants := make([]ParticipantResponse, len(resp.Participants))
	for i, p := range resp.Participants {
		participants[i] = participantToResponse(p)
	}

	h.writeJSON(w, http.StatusOK, ParticipantsResponse{
		Participants: participants,
	})
}

// MuteParticipant godoc
// @Summary Mute/unmute a participant
// @Description Mutes or unmutes a participant in a conference
// @Tags voice
// @Accept json
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Param userID path string true "User ID to mute"
// @Param request body MuteParticipantRequest true "Mute options"
// @Success 200 {object} ParticipantResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants/{userID}/mute [post]
func (h *VoiceHandler) MuteParticipant(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")
	targetUserID := chi.URLParam(r, "userID")

	var req MuteParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	participant, err := h.voiceClient.MuteParticipant(r.Context(), conferenceID, actorUserID.String(), targetUserID, req.Mute)
	if err != nil {
		h.log.Error("failed to mute participant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to mute participant")
		return
	}

	h.writeJSON(w, http.StatusOK, participantToResponse(participant))
}

// KickParticipant godoc
// @Summary Kick a participant
// @Description Removes a participant from a conference (moderator only)
// @Tags voice
// @Param conferenceID path string true "Conference ID"
// @Param userID path string true "User ID to kick"
// @Success 204
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants/{userID}/kick [post]
func (h *VoiceHandler) KickParticipant(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")
	targetUserID := chi.URLParam(r, "userID")

	if err := h.voiceClient.KickParticipant(r.Context(), conferenceID, actorUserID.String(), targetUserID); err != nil {
		h.log.Error("failed to kick participant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to kick participant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EndConference godoc
// @Summary End a conference
// @Description Ends a conference (creator only)
// @Tags voice
// @Param conferenceID path string true "Conference ID"
// @Success 204
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/end [post]
func (h *VoiceHandler) EndConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	if err := h.voiceClient.EndConference(r.Context(), conferenceID, userID.String()); err != nil {
		h.log.Error("failed to end conference", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to end conference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// InitiateCall godoc
// @Summary Initiate a call
// @Description Starts a 1-on-1 call with another user
// @Tags voice
// @Accept json
// @Produce json
// @Param request body InitiateCallRequest true "Call request"
// @Success 201 {object} CallResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/calls [post]
func (h *VoiceHandler) InitiateCall(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req InitiateCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CalleeID == "" {
		h.writeError(w, http.StatusBadRequest, "callee_id is required")
		return
	}

	call, err := h.voiceClient.InitiateCall(r.Context(), callerID.String(), req.CalleeID, req.ChatID)
	if err != nil {
		h.log.Error("failed to initiate call", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to initiate call")
		return
	}

	h.writeJSON(w, http.StatusCreated, callToResponse(call))
}

// AnswerCall godoc
// @Summary Answer a call
// @Description Answers an incoming call
// @Tags voice
// @Produce json
// @Param callID path string true "Call ID"
// @Success 200 {object} CallResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/calls/{callID}/answer [post]
func (h *VoiceHandler) AnswerCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	callID := chi.URLParam(r, "callID")

	call, err := h.voiceClient.AnswerCall(r.Context(), callID, userID.String())
	if err != nil {
		h.log.Error("failed to answer call", "error", err, "callID", callID)
		h.writeError(w, http.StatusInternalServerError, "failed to answer call")
		return
	}

	h.writeJSON(w, http.StatusOK, callToResponse(call))
}

// HangupCall godoc
// @Summary Hangup a call
// @Description Ends a call
// @Tags voice
// @Param callID path string true "Call ID"
// @Success 204
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/calls/{callID}/hangup [post]
func (h *VoiceHandler) HangupCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	callID := chi.URLParam(r, "callID")

	if err := h.voiceClient.HangupCall(r.Context(), callID, userID.String()); err != nil {
		h.log.Error("failed to hangup call", "error", err, "callID", callID)
		h.writeError(w, http.StatusInternalServerError, "failed to hangup call")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetCallHistory godoc
// @Summary Get call history
// @Description Returns call history for the current user
// @Tags voice
// @Produce json
// @Param limit query int false "Number of results to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} CallHistoryResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/calls/history [get]
func (h *VoiceHandler) GetCallHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	resp, err := h.voiceClient.GetCallHistory(r.Context(), userID.String(), int32(limit), int32(offset))
	if err != nil {
		h.log.Error("failed to get call history", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get call history")
		return
	}

	calls := make([]CallResponse, len(resp.Calls))
	for i, call := range resp.Calls {
		calls[i] = callToResponse(call)
	}

	h.writeJSON(w, http.StatusOK, CallHistoryResponse{
		Calls: calls,
		Total: int(resp.Total),
	})
}

// GetVertoCredentials godoc
// @Summary Get Verto credentials
// @Description Generates temporary credentials for Verto WebSocket connection
// @Tags voice
// @Produce json
// @Success 200 {object} VertoCredentialsResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/credentials [get]
func (h *VoiceHandler) GetVertoCredentials(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	creds, err := h.voiceClient.GetVertoCredentials(r.Context(), userID.String())
	if err != nil {
		h.log.Error("failed to get verto credentials", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to get verto credentials")
		return
	}

	iceServers := make([]IceServerResponse, len(creds.IceServers))
	for i, ice := range creds.IceServers {
		iceServers[i] = IceServerResponse{
			URLs:       ice.Urls,
			Username:   ice.Username,
			Credential: ice.Credential,
		}
	}

	h.writeJSON(w, http.StatusOK, VertoCredentialsResponse{
		UserID:     creds.UserId,
		Login:      creds.Login,
		Password:   creds.Password,
		WSUrl:      creds.WsUrl,
		IceServers: iceServers,
		ExpiresAt:  creds.ExpiresAt,
	})
}

// StartChatCall godoc
// @Summary Start a chat call
// @Description Starts a voice call from a chat room
// @Tags voice
// @Accept json
// @Produce json
// @Param chatID path string true "Chat ID"
// @Param request body StartChatCallRequest false "Optional call parameters"
// @Success 200 {object} ChatCallResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/chats/{chatID}/call [post]
func (h *VoiceHandler) StartChatCall(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatID")

	// Parse optional request body
	var req StartChatCallRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Ignore decode errors - name is optional
			h.log.Debug("could not decode start chat call request", "error", err)
		}
	}

	resp, err := h.voiceClient.StartChatCall(r.Context(), chatID, userID.String(), req.Name)
	if err != nil {
		h.log.Error("failed to start chat call", "error", err, "chatID", chatID)
		h.writeError(w, http.StatusInternalServerError, "failed to start chat call")
		return
	}

	iceServers := make([]IceServerResponse, len(resp.Credentials.IceServers))
	for i, ice := range resp.Credentials.IceServers {
		iceServers[i] = IceServerResponse{
			URLs:       ice.Urls,
			Username:   ice.Username,
			Credential: ice.Credential,
		}
	}

	h.writeJSON(w, http.StatusOK, ChatCallResponse{
		Conference: conferenceToResponse(resp.Conference),
		Credentials: VertoCredentialsResponse{
			UserID:     resp.Credentials.UserId,
			Login:      resp.Credentials.Login,
			Password:   resp.Credentials.Password,
			WSUrl:      resp.Credentials.WsUrl,
			IceServers: iceServers,
			ExpiresAt:  resp.Credentials.ExpiresAt,
		},
	})
}

// ======== Scheduled Events ========

// ScheduleConference godoc
// @Summary Schedule a conference
// @Description Creates a scheduled or recurring conference
// @Tags voice
// @Accept json
// @Produce json
// @Param request body ScheduleConferenceRequest true "Schedule request"
// @Success 201 {object} ConferenceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/schedule [post]
func (h *VoiceHandler) ScheduleConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req ScheduleConferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid scheduled_at format")
		return
	}

	grpcReq := &pb.ScheduleConferenceRequest{
		Name:               req.Name,
		ChatId:             req.ChatID,
		UserId:             userID.String(),
		ScheduledAt:        timestamppb.New(scheduledAt),
		ParticipantUserIds: req.ParticipantIDs,
		MaxMembers:         int32(req.MaxMembers),
		EnableRecording:    req.EnableRecording,
	}

	if req.Recurrence != nil {
		grpcReq.Recurrence = &pb.RecurrenceRule{
			Frequency:  pb.RecurrenceFrequency(pb.RecurrenceFrequency_value["RECURRENCE_FREQUENCY_"+req.Recurrence.Frequency]),
			DaysOfWeek: req.Recurrence.DaysOfWeek,
			DayOfMonth: int32(req.Recurrence.DayOfMonth),
		}
		if req.Recurrence.Until != "" {
			until, _ := time.Parse(time.RFC3339, req.Recurrence.Until)
			grpcReq.Recurrence.Until = timestamppb.New(until)
		}
		if req.Recurrence.Count > 0 {
			grpcReq.Recurrence.Count = int32(req.Recurrence.Count)
		}
	}

	conf, err := h.voiceClient.ScheduleConference(r.Context(), grpcReq)
	if err != nil {
		h.log.Error("failed to schedule conference", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to schedule conference")
		return
	}

	h.writeJSON(w, http.StatusCreated, scheduledConferenceToResponse(conf))
}

// ListScheduledConferences godoc
// @Summary List scheduled conferences
// @Description Lists scheduled conferences for the current user
// @Tags voice
// @Produce json
// @Param upcoming_only query bool false "Only show upcoming conferences"
// @Param limit query int false "Number of results to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} ListScheduledConferencesResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/scheduled [get]
func (h *VoiceHandler) ListScheduledConferences(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	upcomingOnly := r.URL.Query().Get("upcoming_only") == "true"
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	resp, err := h.voiceClient.ListScheduledConferences(r.Context(), &pb.ListScheduledConferencesRequest{
		UserId:       userID.String(),
		UpcomingOnly: upcomingOnly,
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		// If voice service is unavailable, return empty list
		h.log.Warn("voice service unavailable, returning empty scheduled conferences", "error", err)
		h.writeJSON(w, http.StatusOK, ListScheduledConferencesResponse{
			Conferences: []ScheduledConferenceResponse{},
			Total:       0,
		})
		return
	}

	conferences := make([]ScheduledConferenceResponse, len(resp.Conferences))
	for i, conf := range resp.Conferences {
		conferences[i] = scheduledConferenceToResponse(conf)
	}

	h.writeJSON(w, http.StatusOK, ListScheduledConferencesResponse{
		Conferences: conferences,
		Total:       int(resp.Total),
	})
}

// CreateAdHocFromChat godoc
// @Summary Create ad-hoc call from chat
// @Description Creates an ad-hoc call from a chat with selected participants
// @Tags voice
// @Accept json
// @Produce json
// @Param request body CreateAdHocFromChatRequest true "Ad-hoc request"
// @Success 201 {object} ConferenceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/adhoc-chat [post]
func (h *VoiceHandler) CreateAdHocFromChat(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateAdHocFromChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ChatID == "" {
		h.writeError(w, http.StatusBadRequest, "chat_id is required")
		return
	}

	conf, err := h.voiceClient.CreateAdHocFromChat(r.Context(), &pb.CreateAdHocFromChatRequest{
		ChatId:             req.ChatID,
		UserId:             userID.String(),
		ParticipantUserIds: req.ParticipantIDs,
	})
	if err != nil {
		h.log.Error("failed to create ad-hoc call", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create ad-hoc call")
		return
	}

	h.writeJSON(w, http.StatusCreated, scheduledConferenceToResponse(conf))
}

// CreateQuickAdHoc godoc
// @Summary Create quick ad-hoc call
// @Description Creates a quick ad-hoc call without a chat
// @Tags voice
// @Accept json
// @Produce json
// @Param request body CreateQuickAdHocRequest true "Quick ad-hoc request"
// @Success 201 {object} ConferenceResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/quick-adhoc [post]
func (h *VoiceHandler) CreateQuickAdHoc(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateQuickAdHocRequest
	json.NewDecoder(r.Body).Decode(&req) // Optional body

	// Use ScheduleConference with event_type = adhoc
	conf, err := h.voiceClient.ScheduleConference(r.Context(), &pb.ScheduleConferenceRequest{
		Name:        req.Name,
		UserId:      userID.String(),
		ScheduledAt: timestamppb.Now(), // Immediate
		MaxMembers:  50,
	})
	if err != nil {
		h.log.Error("failed to create quick ad-hoc", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create quick ad-hoc call")
		return
	}

	h.writeJSON(w, http.StatusCreated, scheduledConferenceToResponse(conf))
}

// UpdateRSVP godoc
// @Summary Update RSVP status
// @Description Updates the current user's RSVP status for a scheduled conference
// @Tags voice
// @Accept json
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Param request body UpdateRSVPRequest true "RSVP update request"
// @Success 200 {object} ParticipantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/rsvp [put]
func (h *VoiceHandler) UpdateRSVP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	var req UpdateRSVPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rsvpStatus := pb.RSVPStatus_RSVP_STATUS_PENDING
	switch req.Status {
	case "accepted":
		rsvpStatus = pb.RSVPStatus_RSVP_STATUS_ACCEPTED
	case "declined":
		rsvpStatus = pb.RSVPStatus_RSVP_STATUS_DECLINED
	}

	participant, err := h.voiceClient.UpdateRSVP(r.Context(), &pb.UpdateRSVPRequest{
		ConferenceId: conferenceID,
		UserId:       userID.String(),
		RsvpStatus:   rsvpStatus,
	})
	if err != nil {
		h.log.Error("failed to update RSVP", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusInternalServerError, "failed to update RSVP")
		return
	}

	h.writeJSON(w, http.StatusOK, scheduledParticipantToResponse(participant))
}

// UpdateParticipantRole godoc
// @Summary Update participant role
// @Description Updates a participant's role in a scheduled conference
// @Tags voice
// @Accept json
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Param userID path string true "User ID"
// @Param request body UpdateParticipantRoleRequest true "Role update request"
// @Success 200 {object} ParticipantResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants/{userID}/role [put]
func (h *VoiceHandler) UpdateParticipantRole(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")
	targetUserID := chi.URLParam(r, "userID")

	var req UpdateParticipantRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newRole := pb.ConferenceRole_CONFERENCE_ROLE_PARTICIPANT
	switch req.Role {
	case "originator":
		newRole = pb.ConferenceRole_CONFERENCE_ROLE_ORIGINATOR
	case "moderator":
		newRole = pb.ConferenceRole_CONFERENCE_ROLE_MODERATOR
	case "speaker":
		newRole = pb.ConferenceRole_CONFERENCE_ROLE_SPEAKER
	case "assistant":
		newRole = pb.ConferenceRole_CONFERENCE_ROLE_ASSISTANT
	}

	participant, err := h.voiceClient.UpdateParticipantRole(r.Context(), &pb.UpdateParticipantRoleRequest{
		ConferenceId: conferenceID,
		ActorUserId:  actorUserID.String(),
		TargetUserId: targetUserID,
		NewRole:      newRole,
	})
	if err != nil {
		h.log.Error("failed to update role", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to update role")
		return
	}

	h.writeJSON(w, http.StatusOK, scheduledParticipantToResponse(participant))
}

// AddParticipants godoc
// @Summary Add participants to conference
// @Description Adds participants to a scheduled conference
// @Tags voice
// @Accept json
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Param request body AddParticipantsRequest true "Add participants request"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants [post]
func (h *VoiceHandler) AddParticipants(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	var req AddParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	defaultRole := pb.ConferenceRole_CONFERENCE_ROLE_PARTICIPANT
	if req.DefaultRole != "" {
		switch req.DefaultRole {
		case "moderator":
			defaultRole = pb.ConferenceRole_CONFERENCE_ROLE_MODERATOR
		case "speaker":
			defaultRole = pb.ConferenceRole_CONFERENCE_ROLE_SPEAKER
		case "assistant":
			defaultRole = pb.ConferenceRole_CONFERENCE_ROLE_ASSISTANT
		}
	}

	_, err := h.voiceClient.AddParticipants(r.Context(), &pb.AddParticipantsRequest{
		ConferenceId: conferenceID,
		ActorUserId:  actorUserID.String(),
		UserIds:      req.UserIDs,
		DefaultRole:  defaultRole,
	})
	if err != nil {
		h.log.Error("failed to add participants", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to add participants")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveParticipant godoc
// @Summary Remove participant from conference
// @Description Removes a participant from a scheduled conference
// @Tags voice
// @Param conferenceID path string true "Conference ID"
// @Param userID path string true "User ID to remove"
// @Success 204
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/participants/{userID} [delete]
func (h *VoiceHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")
	targetUserID := chi.URLParam(r, "userID")

	_, err := h.voiceClient.RemoveParticipant(r.Context(), &pb.RemoveParticipantRequest{
		ConferenceId: conferenceID,
		ActorUserId:  actorUserID.String(),
		TargetUserId: targetUserID,
	})
	if err != nil {
		h.log.Error("failed to remove participant", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to remove participant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetChatConferences godoc
// @Summary Get chat conferences
// @Description Gets conferences for a specific chat
// @Tags voice
// @Produce json
// @Param chatID path string true "Chat ID"
// @Param upcoming_only query bool false "Only show upcoming conferences"
// @Success 200 {object} ChatConferencesResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/chats/{chatID}/conferences [get]
func (h *VoiceHandler) GetChatConferences(w http.ResponseWriter, r *http.Request) {
	chatID := chi.URLParam(r, "chatID")
	upcomingOnly := r.URL.Query().Get("upcoming_only") == "true"

	resp, err := h.voiceClient.GetChatConferences(r.Context(), &pb.GetChatConferencesRequest{
		ChatId:       chatID,
		UpcomingOnly: upcomingOnly,
	})
	if err != nil {
		// If voice service is unavailable, return empty list instead of error
		// This allows the app to work without voice service deployed
		h.log.Warn("voice service unavailable, returning empty conferences", "error", err, "chatID", chatID)
		h.writeJSON(w, http.StatusOK, ChatConferencesResponse{
			Conferences: []ScheduledConferenceResponse{},
		})
		return
	}

	conferences := make([]ScheduledConferenceResponse, len(resp.Conferences))
	for i, conf := range resp.Conferences {
		conferences[i] = scheduledConferenceToResponse(conf)
	}

	h.writeJSON(w, http.StatusOK, ChatConferencesResponse{
		Conferences: conferences,
	})
}

// CancelConference godoc
// @Summary Cancel a scheduled conference
// @Description Cancels a scheduled conference (originator only)
// @Tags voice
// @Param conferenceID path string true "Conference ID"
// @Param cancel_series query bool false "Cancel entire series for recurring"
// @Success 204
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID} [delete]
func (h *VoiceHandler) CancelConference(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")
	cancelSeries := r.URL.Query().Get("cancel_series") == "true"

	_, err := h.voiceClient.CancelConference(r.Context(), &pb.CancelConferenceRequest{
		ConferenceId: conferenceID,
		UserId:       userID.String(),
		CancelSeries: cancelSeries,
	})
	if err != nil {
		h.log.Error("failed to cancel conference", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to cancel conference")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ======== Conference History ========

// ListChatConferenceHistory godoc
// @Summary List conference history for a chat
// @Description Returns history of all conferences in a chat
// @Tags voice
// @Produce json
// @Param chatID path string true "Chat ID"
// @Param limit query int false "Number of results to return" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} ListConferenceHistoryResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/chats/{chatID}/conferences/history [get]
func (h *VoiceHandler) ListChatConferenceHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatID")
	limit := h.parseIntParam(r, "limit", 20)
	offset := h.parseIntParam(r, "offset", 0)

	resp, err := h.voiceClient.ListChatConferenceHistory(r.Context(), chatID, userID.String(), int32(limit), int32(offset))
	if err != nil {
		h.log.Warn("voice service unavailable, returning empty conference history", "error", err, "chatID", chatID)
		h.writeJSON(w, http.StatusOK, ListConferenceHistoryResponse{
			Conferences: []ConferenceHistoryResponse{},
			Total:       0,
		})
		return
	}

	conferences := make([]ConferenceHistoryResponse, len(resp.Conferences))
	for i, conf := range resp.Conferences {
		conferences[i] = conferenceHistoryToResponse(conf)
	}

	h.writeJSON(w, http.StatusOK, ListConferenceHistoryResponse{
		Conferences: conferences,
		Total:       int(resp.Total),
	})
}

// GetConferenceHistory godoc
// @Summary Get conference history details
// @Description Returns detailed history for a specific conference including all participants
// @Tags voice
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Success 200 {object} ConferenceHistoryResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/history [get]
func (h *VoiceHandler) GetConferenceHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	resp, err := h.voiceClient.GetConferenceHistory(r.Context(), conferenceID, userID.String())
	if err != nil {
		h.log.Error("failed to get conference history", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusNotFound, "conference not found")
		return
	}

	h.writeJSON(w, http.StatusOK, conferenceHistoryToResponse(resp))
}

// GetConferenceMessages godoc
// @Summary Get conference messages
// @Description Returns messages sent during a conference
// @Tags voice
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Success 200 {object} ConferenceMessagesResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/messages [get]
func (h *VoiceHandler) GetConferenceMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	resp, err := h.voiceClient.GetConferenceMessages(r.Context(), conferenceID, userID.String())
	if err != nil {
		h.log.Error("failed to get conference messages", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusInternalServerError, "failed to get conference messages")
		return
	}

	messages := make([]ConferenceMessageResponse, len(resp.Messages))
	for i, msg := range resp.Messages {
		messages[i] = ConferenceMessageResponse{
			ID:                msg.Id,
			ChatID:            msg.ChatId,
			SenderID:          msg.SenderId,
			SenderUsername:    msg.SenderUsername,
			SenderDisplayName: msg.SenderDisplayName,
			Content:           msg.Content,
			CreatedAt:         protoTimestampToString(msg.CreatedAt),
		}
	}

	h.writeJSON(w, http.StatusOK, ConferenceMessagesResponse{
		Messages: messages,
	})
}

// GetModeratorActions godoc
// @Summary Get moderator actions
// @Description Returns moderator actions for a conference (moderators/owners only)
// @Tags voice
// @Produce json
// @Param conferenceID path string true "Conference ID"
// @Success 200 {object} ModeratorActionsResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security Bearer
// @Router /voice/conferences/{conferenceID}/moderator-actions [get]
func (h *VoiceHandler) GetModeratorActions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	conferenceID := chi.URLParam(r, "conferenceID")

	resp, err := h.voiceClient.GetModeratorActions(r.Context(), conferenceID, userID.String())
	if err != nil {
		h.log.Error("failed to get moderator actions", "error", err, "conferenceID", conferenceID)
		h.writeError(w, http.StatusForbidden, "access denied or conference not found")
		return
	}

	actions := make([]ModeratorActionResponse, len(resp.Actions))
	for i, action := range resp.Actions {
		actions[i] = ModeratorActionResponse{
			ID:                action.Id,
			ConferenceID:      action.ConferenceId,
			ActorID:           action.ActorId,
			TargetUserID:      action.TargetUserId,
			ActionType:        action.ActionType,
			Details:           action.Details,
			ActorUsername:     action.ActorUsername,
			ActorDisplayName:  action.ActorDisplayName,
			TargetUsername:    action.TargetUsername,
			TargetDisplayName: action.TargetDisplayName,
			CreatedAt:         protoTimestampToString(action.CreatedAt),
		}
	}

	h.writeJSON(w, http.StatusOK, ModeratorActionsResponse{
		Actions: actions,
	})
}

// Helper methods

func (h *VoiceHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *VoiceHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

func (h *VoiceHandler) parseIntParam(r *http.Request, param string, defaultValue int) int {
	valStr := r.URL.Query().Get(param)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}

// Helper functions

func conferenceToResponse(conf *pb.Conference) ConferenceResponse {
	return ConferenceResponse{
		ID:               conf.Id,
		Name:             conf.Name,
		FreeSwitchName:   conf.FreeswitchName,
		ChatID:           conf.ChatId,
		CreatedBy:        conf.CreatedBy,
		Status:           conf.Status.String(),
		MaxMembers:       int(conf.MaxMembers),
		ParticipantCount: int(conf.ParticipantCount),
		IsPrivate:        conf.IsPrivate,
		RecordingPath:    conf.RecordingPath,
		StartedAt:        protoTimestampToString(conf.StartedAt),
		EndedAt:          protoTimestampToString(conf.EndedAt),
		CreatedAt:        protoTimestampToString(conf.CreatedAt),
	}
}

func participantToResponse(p *pb.Participant) ParticipantResponse {
	return ParticipantResponse{
		ID:           p.Id,
		ConferenceID: p.ConferenceId,
		UserID:       p.UserId,
		Status:       p.Status.String(),
		IsMuted:      p.IsMuted,
		IsDeaf:       p.IsDeaf,
		IsSpeaking:   p.IsSpeaking,
		Username:     p.Username,
		DisplayName:  p.DisplayName,
		AvatarURL:    p.AvatarUrl,
		JoinedAt:     protoTimestampToString(p.JoinedAt),
	}
}

func callToResponse(call *pb.Call) CallResponse {
	return CallResponse{
		ID:                call.Id,
		CallerID:          call.CallerId,
		CalleeID:          call.CalleeId,
		ChatID:            call.ChatId,
		ConferenceID:      call.ConferenceId,
		Status:            call.Status.String(),
		Duration:          int(call.Duration),
		CallerUsername:    call.CallerUsername,
		CallerDisplayName: call.CallerDisplayName,
		CalleeUsername:    call.CalleeUsername,
		CalleeDisplayName: call.CalleeDisplayName,
		StartedAt:         protoTimestampToString(call.StartedAt),
		AnsweredAt:        protoTimestampToString(call.AnsweredAt),
		EndedAt:           protoTimestampToString(call.EndedAt),
	}
}

func protoTimestampToString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().Format(time.RFC3339)
}

// Scheduled Events Helper Functions

func scheduledConferenceToResponse(conf *pb.Conference) ScheduledConferenceResponse {
	resp := ScheduledConferenceResponse{
		ID:               conf.Id,
		Name:             conf.Name,
		FreeSwitchName:   conf.FreeswitchName,
		ChatID:           conf.ChatId,
		CreatedBy:        conf.CreatedBy,
		Status:           conf.Status.String(),
		EventType:        conf.EventType.String(),
		MaxMembers:       int(conf.MaxMembers),
		ParticipantCount: int(conf.ParticipantCount),
		AcceptedCount:    int(conf.AcceptedCount),
		DeclinedCount:    int(conf.DeclinedCount),
		SeriesID:         conf.SeriesId,
		ScheduledAt:      protoTimestampToString(conf.ScheduledAt),
		CreatedAt:        protoTimestampToString(conf.CreatedAt),
	}

	if conf.Recurrence != nil {
		resp.Recurrence = &RecurrenceRuleResponse{
			Frequency:  conf.Recurrence.Frequency.String(),
			DaysOfWeek: conf.Recurrence.DaysOfWeek,
			DayOfMonth: int(conf.Recurrence.DayOfMonth),
			Until:      protoTimestampToString(conf.Recurrence.Until),
			Count:      int(conf.Recurrence.Count),
		}
	}

	if len(conf.Participants) > 0 {
		resp.Participants = make([]ScheduledParticipantResponse, len(conf.Participants))
		for i, p := range conf.Participants {
			resp.Participants[i] = scheduledParticipantToResponse(p)
		}
	}

	return resp
}

func scheduledParticipantToResponse(p *pb.Participant) ScheduledParticipantResponse {
	return ScheduledParticipantResponse{
		ID:           p.Id,
		ConferenceID: p.ConferenceId,
		UserID:       p.UserId,
		Status:       p.Status.String(),
		Role:         conferenceRoleToString(p.Role),
		RSVPStatus:   rsvpStatusToString(p.RsvpStatus),
		RSVPAt:       protoTimestampToString(p.RsvpAt),
		Username:     p.Username,
		DisplayName:  p.DisplayName,
		AvatarURL:    p.AvatarUrl,
		JoinedAt:     protoTimestampToString(p.JoinedAt),
	}
}

// conferenceRoleToString converts proto enum to lowercase string
func conferenceRoleToString(role pb.ConferenceRole) string {
	switch role {
	case pb.ConferenceRole_CONFERENCE_ROLE_ORIGINATOR:
		return "originator"
	case pb.ConferenceRole_CONFERENCE_ROLE_MODERATOR:
		return "moderator"
	case pb.ConferenceRole_CONFERENCE_ROLE_SPEAKER:
		return "speaker"
	case pb.ConferenceRole_CONFERENCE_ROLE_ASSISTANT:
		return "assistant"
	case pb.ConferenceRole_CONFERENCE_ROLE_PARTICIPANT:
		return "participant"
	default:
		return "participant"
	}
}

// rsvpStatusToString converts proto enum to lowercase string
func rsvpStatusToString(status pb.RSVPStatus) string {
	switch status {
	case pb.RSVPStatus_RSVP_STATUS_PENDING:
		return "pending"
	case pb.RSVPStatus_RSVP_STATUS_ACCEPTED:
		return "accepted"
	case pb.RSVPStatus_RSVP_STATUS_DECLINED:
		return "declined"
	default:
		return "pending"
	}
}

// conferenceHistoryToResponse converts proto ConferenceHistoryResponse to API response
func conferenceHistoryToResponse(conf *pb.ConferenceHistoryResponse) ConferenceHistoryResponse {
	resp := ConferenceHistoryResponse{
		ID:               conf.Id,
		Name:             conf.Name,
		ChatID:           conf.ChatId,
		Status:           conf.Status.String(),
		StartedAt:        protoTimestampToString(conf.StartedAt),
		EndedAt:          protoTimestampToString(conf.EndedAt),
		CreatedAt:        protoTimestampToString(conf.CreatedAt),
		ParticipantCount: int(conf.ParticipantCount),
		ThreadID:         conf.ThreadId,
	}

	if len(conf.AllParticipants) > 0 {
		resp.AllParticipants = make([]ParticipantHistoryResponse, len(conf.AllParticipants))
		for i, p := range conf.AllParticipants {
			sessions := make([]ParticipantSessionResponse, len(p.Sessions))
			for j, s := range p.Sessions {
				sessions[j] = ParticipantSessionResponse{
					JoinedAt: protoTimestampToString(s.JoinedAt),
					LeftAt:   protoTimestampToString(s.LeftAt),
					Status:   s.Status.String(),
					Role:     conferenceRoleToString(s.Role),
				}
			}
			resp.AllParticipants[i] = ParticipantHistoryResponse{
				UserID:      p.UserId,
				Username:    p.Username,
				DisplayName: p.DisplayName,
				Sessions:    sessions,
			}
		}
	}

	return resp
}
