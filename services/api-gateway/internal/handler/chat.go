package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/icegreg/chat-smpl/pkg/logger"
	pb "github.com/icegreg/chat-smpl/proto/chat"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/files"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

type ChatHandler struct {
	chatClient  *grpc.ChatClient
	filesClient *files.Client
	log         logger.Logger
}

func NewChatHandler(chatClient *grpc.ChatClient, filesClient *files.Client, log logger.Logger) *ChatHandler {
	return &ChatHandler{
		chatClient:  chatClient,
		filesClient: filesClient,
		log:         log,
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
	r.Get("/{chatId}/messages/sync", h.SyncMessages)
	r.Post("/{chatId}/messages", h.SendMessage)
	r.Put("/messages/{messageId}", h.UpdateMessage)
	r.Delete("/messages/{messageId}", h.DeleteMessage)

	// Reaction routes
	r.Post("/messages/{messageId}/reactions", h.AddReaction)
	r.Delete("/messages/{messageId}/reactions/{emoji}", h.RemoveReaction)

	// Thread routes
	r.Get("/{chatId}/threads", h.ListChatThreads)
	r.Post("/{chatId}/threads", h.CreateChatThread)
	r.Get("/threads/{threadId}", h.GetThreadByID)
	r.Post("/threads/{threadId}/archive", h.ArchiveThreadByID)
	r.Get("/threads/{threadId}/messages", h.GetThreadMessages)
	r.Post("/threads/{threadId}/participants", h.AddThreadParticipantHandler)
	r.Delete("/threads/{threadId}/participants/{userId}", h.RemoveThreadParticipantHandler)
	r.Get("/threads/{threadId}/participants", h.ListThreadParticipantsHandler)
	// Subthread routes
	r.Get("/threads/{threadId}/subthreads", h.ListSubthreads)
	r.Post("/threads/{threadId}/subthreads", h.CreateSubthread)
	// Reply thread (from message) - kept for backwards compatibility
	r.Post("/messages/{messageId}/thread", h.CreateThread)

	// Favorites and Archive
	r.Post("/{chatId}/favorite", h.AddToFavorites)
	r.Delete("/{chatId}/favorite", h.RemoveFromFavorites)
	r.Post("/{chatId}/archive", h.ArchiveChat)
	r.Delete("/{chatId}/archive", h.UnarchiveChat)

	// Typing indicator
	r.Post("/{chatId}/typing", h.SendTypingIndicator)

	// Forward message
	r.Post("/messages/{messageId}/forward", h.ForwardMessage)

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

	// If message has file attachments, grant permissions to chat participants
	if len(req.FileLinkIDs) > 0 {
		if err := h.grantFilePermissionsToParticipants(ctx, chatID, req.FileLinkIDs, userID.String()); err != nil {
			h.log.Error("failed to grant file permissions", "error", err, "chatId", chatID)
			// Continue anyway - permissions can be added later if needed
		}
	}

	message, err := h.chatClient.SendMessage(ctx, chatID, userID.String(), req.Content, req.ParentID, req.FileLinkIDs)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	// Enrich message with file attachments
	enrichedMessages := h.enrichMessagesWithFiles(ctx, []*pb.Message{message})
	h.respondJSON(w, http.StatusCreated, enrichedMessages[0])
}

// grantFilePermissionsToParticipants grants file permissions to all chat participants
func (h *ChatHandler) grantFilePermissionsToParticipants(ctx context.Context, chatID string, fileLinkIDs []string, uploaderID string) error {
	// Get all participants of the chat
	// We use a high count to get all participants in one request
	participantsResp, err := h.chatClient.ListParticipants(ctx, chatID, 1, 1000)
	if err != nil {
		return err
	}

	if len(participantsResp.Participants) == 0 {
		return nil
	}

	// Extract participant user IDs
	userIDs := make([]string, 0, len(participantsResp.Participants))
	for _, p := range participantsResp.Participants {
		userIDs = append(userIDs, p.UserId)
	}

	// Grant permissions
	return h.filesClient.GrantPermissions(ctx, fileLinkIDs, userIDs, uploaderID)
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

	// Enrich messages with file attachments
	messages := h.enrichMessagesWithFiles(ctx, resp.Messages)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages":   messages,
		"pagination": resp.Pagination,
	})
}

// SyncMessages returns messages after a specific seq_num for reliable sync after reconnect
// GET /api/chats/{chatId}/messages/sync?after_seq=123&limit=100
func (h *ChatHandler) SyncMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")
	afterSeqNum, _ := strconv.ParseInt(r.URL.Query().Get("after_seq"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	resp, err := h.chatClient.SyncMessages(ctx, chatID, userID.String(), afterSeqNum, int32(limit))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	// Enrich messages with file attachments
	messages := h.enrichMessagesWithFiles(ctx, resp.Messages)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
		"has_more": resp.HasMore,
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

// ListChatThreads returns threads for a chat
// GET /api/chats/{chatId}/threads
func (h *ChatHandler) ListChatThreads(w http.ResponseWriter, r *http.Request) {
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
		count = 20
	}

	resp, err := h.chatClient.ListThreads(ctx, chatID, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"threads":    resp.Threads,
		"pagination": resp.Pagination,
	})
}

// CreateChatThread creates a new thread in a chat
// POST /api/chats/{chatId}/threads
func (h *ChatHandler) CreateChatThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	chatID := chi.URLParam(r, "chatId")

	var req struct {
		ParentMessageID        string `json:"parent_message_id,omitempty"`
		ThreadType             string `json:"thread_type"`
		Title                  string `json:"title,omitempty"`
		RestrictedParticipants bool   `json:"restricted_participants"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Convert thread type string to enum
	threadType := pb.ThreadType_THREAD_TYPE_USER
	if req.ThreadType == "system" {
		threadType = pb.ThreadType_THREAD_TYPE_SYSTEM
	}

	var parentMsgID *string
	if req.ParentMessageID != "" {
		parentMsgID = &req.ParentMessageID
	}

	var title *string
	if req.Title != "" {
		title = &req.Title
	}

	thread, err := h.chatClient.CreateThread(ctx, chatID, parentMsgID, threadType, title, userID.String(), req.RestrictedParticipants)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, thread)
}

// GetThreadByID returns a specific thread
// GET /api/chats/threads/{threadId}
func (h *ChatHandler) GetThreadByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "threadId")

	thread, err := h.chatClient.GetThread(ctx, threadID, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, thread)
}

// ArchiveThreadByID archives a thread
// POST /api/chats/threads/{threadId}/archive
func (h *ChatHandler) ArchiveThreadByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "threadId")

	thread, err := h.chatClient.ArchiveThread(ctx, threadID, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, thread)
}

// CreateThread creates a reply thread from message (backward compatibility)
// POST /api/chats/messages/{messageId}/thread
func (h *ChatHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	// Get message to find chat ID
	message, err := h.chatClient.GetMessage(ctx, messageID, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	// Create thread with parent message
	thread, err := h.chatClient.CreateThread(ctx, message.ChatId, &messageID, pb.ThreadType_THREAD_TYPE_USER, nil, userID.String(), false)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, thread)
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

	threadID := chi.URLParam(r, "threadId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if page <= 0 {
		page = 1
	}
	if count <= 0 {
		count = 50
	}

	resp, err := h.chatClient.ListThreadMessages(ctx, threadID, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	// Enrich messages with file attachments
	messages := h.enrichMessagesWithFiles(ctx, resp.Messages)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages":   messages,
		"pagination": resp.Pagination,
	})
}

// AddThreadParticipantHandler adds a participant to a thread
// POST /api/chats/threads/{threadId}/participants
func (h *ChatHandler) AddThreadParticipantHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "threadId")

	var req struct {
		UserID string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.chatClient.AddThreadParticipant(ctx, threadID, req.UserID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RemoveThreadParticipantHandler removes a participant from a thread
// DELETE /api/chats/threads/{threadId}/participants/{userId}
func (h *ChatHandler) RemoveThreadParticipantHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "threadId")
	targetUserID := chi.URLParam(r, "userId")

	if err := h.chatClient.RemoveThreadParticipant(ctx, threadID, targetUserID, userID.String()); err != nil {
		h.handleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListThreadParticipantsHandler returns thread participants
// GET /api/chats/threads/{threadId}/participants
func (h *ChatHandler) ListThreadParticipantsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	threadID := chi.URLParam(r, "threadId")

	resp, err := h.chatClient.ListThreadParticipants(ctx, threadID)
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"participants": resp.Participants,
	})
}

// ListSubthreads returns subthreads of a thread
// GET /api/chats/threads/{threadId}/subthreads
func (h *ChatHandler) ListSubthreads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	parentThreadID := chi.URLParam(r, "threadId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if count <= 0 {
		count = 20
	}

	resp, err := h.chatClient.ListSubthreads(ctx, parentThreadID, userID.String(), int32(page), int32(count))
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	total := int32(0)
	if resp.Pagination != nil {
		total = resp.Pagination.Total
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"threads": resp.Threads,
		"total":   total,
	})
}

// CreateSubthread creates a subthread under a parent thread
// POST /api/chats/threads/{threadId}/subthreads
func (h *ChatHandler) CreateSubthread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	parentThreadID := chi.URLParam(r, "threadId")

	var req struct {
		Title      string `json:"title"`
		ThreadType string `json:"thread_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	threadType := pb.ThreadType_THREAD_TYPE_USER
	if req.ThreadType == "system" {
		threadType = pb.ThreadType_THREAD_TYPE_SYSTEM
	}

	thread, err := h.chatClient.CreateSubthread(ctx, parentThreadID, req.Title, threadType, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, thread)
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

// ForwardMessage forwards a message to another chat
// POST /api/chats/messages/{messageId}/forward
func (h *ChatHandler) ForwardMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Check role - guest cannot forward messages
	role, _ := middleware.GetUserRole(ctx)
	if role == "guest" {
		h.respondError(w, http.StatusForbidden, "guests cannot forward messages")
		return
	}

	messageID := chi.URLParam(r, "messageId")

	var req struct {
		TargetChatID string `json:"target_chat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TargetChatID == "" {
		h.respondError(w, http.StatusBadRequest, "target_chat_id is required")
		return
	}

	message, err := h.chatClient.ForwardMessage(ctx, messageID, req.TargetChatID, userID.String())
	if err != nil {
		h.handleGRPCError(w, err)
		return
	}

	// Enrich message with file attachments
	enrichedMessages := h.enrichMessagesWithFiles(ctx, []*pb.Message{message})
	h.respondJSON(w, http.StatusCreated, enrichedMessages[0])
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

// MessageWithAttachments extends pb.Message with file attachments
type MessageWithAttachments struct {
	*pb.Message
	FileAttachments []files.FileAttachment `json:"file_attachments,omitempty"`
}

// MarshalJSON implements custom JSON marshaling to merge protobuf fields with file_attachments
func (m MessageWithAttachments) MarshalJSON() ([]byte, error) {
	// First marshal the embedded protobuf message
	pbJSON, err := json.Marshal(m.Message)
	if err != nil {
		return nil, err
	}

	// If no file attachments, return the protobuf JSON as-is
	if len(m.FileAttachments) == 0 {
		return pbJSON, nil
	}

	// Unmarshal into a map to add file_attachments
	var result map[string]interface{}
	if err := json.Unmarshal(pbJSON, &result); err != nil {
		return nil, err
	}

	// Add file attachments
	result["file_attachments"] = m.FileAttachments

	return json.Marshal(result)
}

// enrichMessagesWithFiles fetches file metadata and adds it to messages
func (h *ChatHandler) enrichMessagesWithFiles(ctx context.Context, messages []*pb.Message) []MessageWithAttachments {
	result := make([]MessageWithAttachments, len(messages))

	// Collect all file link IDs
	allLinkIDs := make([]string, 0)
	for _, msg := range messages {
		allLinkIDs = append(allLinkIDs, msg.FileLinkIds...)
	}

	h.log.Info("enrichMessagesWithFiles", "messages_count", len(messages), "file_link_ids_count", len(allLinkIDs), "link_ids", allLinkIDs)

	// Fetch file metadata in batch
	var fileMap map[string]files.FileAttachment
	if len(allLinkIDs) > 0 {
		fileAttachments, err := h.filesClient.GetFilesByLinkIDs(ctx, allLinkIDs)
		if err != nil {
			h.log.Error("failed to fetch file attachments", "error", err)
		} else {
			h.log.Info("fetched file attachments", "count", len(fileAttachments))
			fileMap = make(map[string]files.FileAttachment)
			for _, f := range fileAttachments {
				fileMap[f.LinkID] = f
			}
		}
	}

	// Enrich messages
	for i, msg := range messages {
		result[i] = MessageWithAttachments{
			Message: msg,
		}

		if len(msg.FileLinkIds) > 0 && fileMap != nil {
			attachments := make([]files.FileAttachment, 0, len(msg.FileLinkIds))
			for _, linkID := range msg.FileLinkIds {
				if attachment, ok := fileMap[linkID]; ok {
					attachments = append(attachments, attachment)
				}
			}
			result[i].FileAttachments = attachments
		}
	}

	return result
}
