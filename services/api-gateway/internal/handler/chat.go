package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/icegreg/chat-smpl/pkg/logger"
	pb "github.com/icegreg/chat-smpl/proto/chat"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

type ChatHandler struct {
	chatClient *grpc.ChatClient
	log        logger.Logger
}

func NewChatHandler(chatClient *grpc.ChatClient, log logger.Logger) *ChatHandler {
	return &ChatHandler{
		chatClient: chatClient,
		log:        log,
	}
}

func (h *ChatHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Chat routes
	r.Post("/", h.CreateChat)
	r.Get("/", h.ListChats)
	r.Get("/{chatId}", h.GetChat)
	r.Put("/{chatId}", h.UpdateChat)
	r.Delete("/{chatId}", h.DeleteChat)

	// Participant routes
	r.Get("/{chatId}/participants", h.GetParticipants)
	r.Post("/{chatId}/participants", h.AddParticipant)
	r.Delete("/{chatId}/participants/{userId}", h.RemoveParticipant)
	r.Put("/{chatId}/participants/{userId}/role", h.UpdateParticipantRole)

	// Message routes
	r.Get("/{chatId}/messages", h.GetMessages)
	r.Post("/{chatId}/messages", h.SendMessage)
	r.Put("/messages/{messageId}", h.UpdateMessage)
	r.Delete("/messages/{messageId}", h.DeleteMessage)

	// Reaction routes
	r.Post("/messages/{messageId}/reactions", h.AddReaction)
	r.Delete("/messages/{messageId}/reactions/{emoji}", h.RemoveReaction)

	// Thread routes
	r.Post("/messages/{messageId}/thread", h.CreateThread)
	r.Get("/threads/{threadId}/messages", h.GetThreadMessages)

	// Favorites and Archive
	r.Post("/{chatId}/favorite", h.AddToFavorites)
	r.Delete("/{chatId}/favorite", h.RemoveFromFavorites)
	r.Post("/{chatId}/archive", h.ArchiveChat)
	r.Delete("/{chatId}/archive", h.UnarchiveChat)

	// Typing indicator
	r.Post("/{chatId}/typing", h.SendTypingIndicator)

	return r
}

// CreateChat creates a new chat
// POST /api/chats
func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Type           string   `json:"type"`
		Name           string   `json:"name"`
		ParticipantIDs []string `json:"participant_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert type string to ChatType enum
	chatType := pb.ChatType_CHAT_TYPE_GROUP
	switch req.Type {
	case "private":
		chatType = pb.ChatType_CHAT_TYPE_PRIVATE
	case "channel":
		chatType = pb.ChatType_CHAT_TYPE_CHANNEL
	}

	chat, err := h.chatClient.CreateChat(ctx, userID.String(), chatType, req.Name, req.ParticipantIDs)
	if err != nil {
		h.log.Error("failed to create chat", "error", err)
		h.respondError(w, http.StatusInternalServerError, "failed to create chat")
		return
	}

	// Events are now published via RabbitMQ -> websocket-service -> Centrifugo

	h.respondJSON(w, http.StatusCreated, chat)
}

// GetChat returns chat details
// GET /api/chats/{chatId}
func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	chat, err := h.chatClient.GetChat(ctx, chatID, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, chat)
}

// ListChats returns list of user's chats
// GET /api/chats
func (h *ChatHandler) ListChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 20
	}

	resp, err := h.chatClient.ListChats(ctx, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"chats":      resp.Chats,
		"pagination": resp.Pagination,
	})
}

// UpdateChat updates chat details
// PUT /api/chats/{chatId}
func (h *ChatHandler) UpdateChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	chat, err := h.chatClient.UpdateChat(ctx, chatID, userID.String(), req.Name)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, chat)
}

// DeleteChat deletes a chat
// DELETE /api/chats/{chatId}
func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	if err := h.chatClient.DeleteChat(ctx, chatID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetParticipants returns chat participants
// GET /api/chats/{chatId}/participants
func (h *ChatHandler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 50
	}

	resp, err := h.chatClient.ListParticipants(ctx, chatID, int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"participants": resp.Participants,
		"pagination":   resp.Pagination,
	})
}

// AddParticipant adds a participant to chat
// POST /api/chats/{chatId}/participants
func (h *ChatHandler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert role string to ParticipantRole enum
	role := pb.ParticipantRole_PARTICIPANT_ROLE_MEMBER
	switch req.Role {
	case "admin":
		role = pb.ParticipantRole_PARTICIPANT_ROLE_ADMIN
	case "readonly":
		role = pb.ParticipantRole_PARTICIPANT_ROLE_READONLY
	}

	participant, err := h.chatClient.AddParticipant(ctx, chatID, req.UserID, userID.String(), role)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, participant)
}

// RemoveParticipant removes a participant from chat
// DELETE /api/chats/{chatId}/participants/{userId}
func (h *ChatHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")
	targetUserID := chi.URLParam(r, "userId")

	if err := h.chatClient.RemoveParticipant(ctx, chatID, userID.String(), targetUserID); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateParticipantRole updates participant's role
// PUT /api/chats/{chatId}/participants/{userId}/role
func (h *ChatHandler) UpdateParticipantRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")
	targetUserID := chi.URLParam(r, "userId")

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert role string to ParticipantRole enum
	role := pb.ParticipantRole_PARTICIPANT_ROLE_MEMBER
	switch req.Role {
	case "admin":
		role = pb.ParticipantRole_PARTICIPANT_ROLE_ADMIN
	case "readonly":
		role = pb.ParticipantRole_PARTICIPANT_ROLE_READONLY
	}

	participant, err := h.chatClient.UpdateParticipantRole(ctx, chatID, targetUserID, userID.String(), role)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, participant)
}

// SendMessage sends a message
// POST /api/chats/{chatId}/messages
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check role - guest cannot send messages
	role, _ := middleware.GetUserRole(ctx)
	if role == "guest" {
		h.respondError(w, http.StatusForbidden, "guests cannot send messages")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	var req struct {
		Content     string   `json:"content"`
		ParentID    string   `json:"parent_id,omitempty"`
		FileLinkIDs []string `json:"file_link_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	message, err := h.chatClient.SendMessage(ctx, chatID, userID.String(), req.Content, req.ParentID, req.FileLinkIDs)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, message)
}

// GetMessages returns chat messages
// GET /api/chats/{chatId}/messages
func (h *ChatHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 50
	}

	resp, err := h.chatClient.ListMessages(ctx, chatID, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages":   resp.Messages,
		"pagination": resp.Pagination,
	})
}

// UpdateMessage updates a message
// PUT /api/chats/messages/{messageId}
func (h *ChatHandler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	message, err := h.chatClient.UpdateMessage(ctx, messageID, userID.String(), req.Content)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, message)
}

// DeleteMessage deletes a message
// DELETE /api/chats/messages/{messageId}
func (h *ChatHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	if err := h.chatClient.DeleteMessage(ctx, messageID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddReaction adds a reaction to message
// POST /api/chats/messages/{messageId}/reactions
func (h *ChatHandler) AddReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	var req struct {
		Emoji string `json:"emoji"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatClient.AddReaction(ctx, messageID, userID.String(), req.Emoji); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveReaction removes a reaction from message
// DELETE /api/chats/messages/{messageId}/reactions/{emoji}
func (h *ChatHandler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "messageId")
	emoji := chi.URLParam(r, "emoji")

	if err := h.chatClient.RemoveReaction(ctx, messageID, userID.String(), emoji); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateThread creates a thread from message (sends a reply message)
// POST /api/chats/messages/{messageId}/thread
func (h *ChatHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	// Thread creation is done by sending a message with parent_id
	h.respondError(w, http.StatusNotImplemented, "use POST /messages with parent_id instead")
}

// GetThreadMessages returns thread messages
// GET /api/chats/threads/{threadId}/messages
func (h *ChatHandler) GetThreadMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	parentID := chi.URLParam(r, "threadId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 50
	}

	resp, err := h.chatClient.GetThreadMessages(ctx, parentID, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages":   resp.Messages,
		"pagination": resp.Pagination,
	})
}

// AddToFavorites adds chat to favorites
// POST /api/chats/{chatId}/favorite
func (h *ChatHandler) AddToFavorites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	if err := h.chatClient.AddToFavorites(ctx, chatID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveFromFavorites removes chat from favorites
// DELETE /api/chats/{chatId}/favorite
func (h *ChatHandler) RemoveFromFavorites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	if err := h.chatClient.RemoveFromFavorites(ctx, chatID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ArchiveChat archives a chat
// POST /api/chats/{chatId}/archive
func (h *ChatHandler) ArchiveChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	if err := h.chatClient.ArchiveChat(ctx, chatID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// UnarchiveChat unarchives a chat
// DELETE /api/chats/{chatId}/archive
func (h *ChatHandler) UnarchiveChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	if err := h.chatClient.UnarchiveChat(ctx, chatID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SendTypingIndicator sends typing status
// POST /api/chats/{chatId}/typing
func (h *ChatHandler) SendTypingIndicator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	var req struct {
		IsTyping bool `json:"is_typing"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Send typing via gRPC - chat-service will publish to RabbitMQ
	if err := h.chatClient.SendTyping(ctx, chatID, userID.String(), req.IsTyping); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Helper methods

func (h *ChatHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (h *ChatHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

func (h *ChatHandler) handleGRPCError(w http.ResponseWriter, err error) {
	h.log.Error("gRPC error", "error", err)
	// Parse gRPC status codes and convert to HTTP
	errStr := err.Error()
	switch {
	case contains(errStr, "not found"):
		h.respondError(w, http.StatusNotFound, "resource not found")
	case contains(errStr, "permission denied"), contains(errStr, "access denied"):
		h.respondError(w, http.StatusForbidden, "access denied")
	case contains(errStr, "invalid"):
		h.respondError(w, http.StatusBadRequest, "invalid request")
	default:
		h.respondError(w, http.StatusInternalServerError, "internal error")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
