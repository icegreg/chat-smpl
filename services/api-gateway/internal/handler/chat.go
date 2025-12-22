package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/icegreg/chat-smpl/pkg/logger"
	pb "github.com/icegreg/chat-smpl/proto/chat"
	orgpb "github.com/icegreg/chat-smpl/proto/org"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/files"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

type ChatHandler struct {
	chatClient  *grpc.ChatClient
	filesClient *files.Client
	orgClient   *grpc.OrgClient
	log         logger.Logger
}

func NewChatHandler(chatClient *grpc.ChatClient, filesClient *files.Client, orgClient *grpc.OrgClient, log logger.Logger) *ChatHandler {
	return &ChatHandler{
		chatClient:  chatClient,
		filesClient: filesClient,
		orgClient:   orgClient,
		log:         log,
	}
}

// ParticipantWithOrg represents a chat participant enriched with org info
type ParticipantWithOrg struct {
	*pb.ChatParticipant
	OrgInfo *orgpb.UserOrgInfo `json:"org_info,omitempty"`
}

// enrichParticipantsWithOrg enriches participants with organization info
func (h *ChatHandler) enrichParticipantsWithOrg(ctx context.Context, participants []*pb.ChatParticipant) []ParticipantWithOrg {
	if h.orgClient == nil || len(participants) == 0 {
		result := make([]ParticipantWithOrg, len(participants))
		for i, p := range participants {
			result[i] = ParticipantWithOrg{ChatParticipant: p}
		}
		return result
	}

	// Collect user IDs
	userIDs := make([]string, len(participants))
	for i, p := range participants {
		userIDs[i] = p.UserId
	}

	// Get org info for all users in batch
	orgInfos, err := h.orgClient.GetUsersOrgInfoBatch(ctx, userIDs)
	if err != nil {
		h.log.Warn("failed to get org info for participants", "error", err)
		// Return participants without org info on error
		result := make([]ParticipantWithOrg, len(participants))
		for i, p := range participants {
			result[i] = ParticipantWithOrg{ChatParticipant: p}
		}
		return result
	}

	// Create a map for quick lookup
	orgInfoMap := make(map[string]*orgpb.UserOrgInfo)
	for _, info := range orgInfos {
		orgInfoMap[info.UserId] = info
	}

	// Enrich participants
	result := make([]ParticipantWithOrg, len(participants))
	for i, p := range participants {
		result[i] = ParticipantWithOrg{
			ChatParticipant: p,
			OrgInfo:         orgInfoMap[p.UserId],
		}
	}

	return result
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

// CreateChat godoc
// @Summary Create a new chat
// @Description Creates a new chat room (private, group, or channel)
// @Tags chats
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateChatRequest true "Chat creation data"
// @Success 201 {object} ChatResponse "Chat created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /chats [post]
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

// GetChat godoc
// @Summary Get chat by ID
// @Description Returns detailed information about a specific chat
// @Tags chats
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 200 {object} ChatResponse "Chat details"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId} [get]
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

// ListChats godoc
// @Summary List user's chats
// @Description Returns paginated list of chats the user participates in
// @Tags chats
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(20)
// @Success 200 {object} ChatListResponse "List of chats"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /chats [get]
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

// UpdateChat godoc
// @Summary Update chat
// @Description Updates chat details (name, description)
// @Tags chats
// @Accept json
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param request body UpdateChatRequest true "Chat update data"
// @Success 200 {object} ChatResponse "Updated chat"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId} [put]
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

// DeleteChat godoc
// @Summary Delete chat
// @Description Deletes a chat and all its messages
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 204 "Chat deleted"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId} [delete]
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

// GetParticipants godoc
// @Summary List chat participants
// @Description Returns paginated list of chat participants
// @Tags chats
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(50)
// @Success 200 {object} map[string]interface{} "Participants list with pagination"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/participants [get]
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

	// Enrich participants with org info
	enrichedParticipants := h.enrichParticipantsWithOrg(ctx, resp.Participants)

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"participants": enrichedParticipants,
		"pagination":   resp.Pagination,
	})
}

// AddParticipant godoc
// @Summary Add participant to chat
// @Description Adds a new participant to the chat with specified role
// @Tags chats
// @Accept json
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param request body AddMembersRequest true "Participant data"
// @Success 201 {object} map[string]interface{} "Participant added"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/participants [post]
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

// RemoveParticipant godoc
// @Summary Remove participant from chat
// @Description Removes a participant from the chat
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param userId path string true "User ID to remove"
// @Success 204 "Participant removed"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Chat or user not found"
// @Router /chats/{chatId}/participants/{userId} [delete]
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

// UpdateParticipantRole godoc
// @Summary Update participant role
// @Description Updates the role of a chat participant
// @Tags chats
// @Accept json
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param userId path string true "User ID"
// @Param request body map[string]string true "Role update data"
// @Success 200 {object} map[string]interface{} "Updated participant"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Chat or user not found"
// @Router /chats/{chatId}/participants/{userId}/role [put]
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

// SendMessage godoc
// @Summary Send a message
// @Description Sends a new message to the chat with optional file attachments and reply references
// @Tags messages
// @Accept json
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param request body SendMessageRequest true "Message data"
// @Success 201 {object} MessageResponse "Message sent"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied (guests cannot send)"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/messages [post]
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
		ReplyToIDs  []string `json:"reply_to_ids,omitempty"`
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

	message, err := h.chatClient.SendMessage(ctx, chatID, userID.String(), req.Content, req.ParentID, req.FileLinkIDs, req.ReplyToIDs)
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

// GetMessages godoc
// @Summary List chat messages
// @Description Returns paginated list of messages in a chat
// @Tags messages
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(50)
// @Success 200 {object} MessageListResponse "Messages list"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/messages [get]
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

// SyncMessages godoc
// @Summary Sync messages
// @Description Returns messages after a specific sequence number for reliable sync after reconnect
// @Tags messages
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param after_seq query int false "Sequence number to start after" default(0)
// @Param limit query int false "Max messages to return" default(100)
// @Success 200 {object} map[string]interface{} "Messages with has_more flag"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/messages/sync [get]
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

// UpdateMessage godoc
// @Summary Update a message
// @Description Updates the content of an existing message
// @Tags messages
// @Accept json
// @Produce json
// @Security Bearer
// @Param messageId path string true "Message ID"
// @Param request body UpdateMessageRequest true "Message update data"
// @Success 200 {object} MessageResponse "Updated message"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Message not found"
// @Router /chats/messages/{messageId} [put]
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

// DeleteMessage godoc
// @Summary Delete a message
// @Description Deletes a message from the chat
// @Tags messages
// @Security Bearer
// @Param messageId path string true "Message ID"
// @Success 204 "Message deleted"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Message not found"
// @Router /chats/messages/{messageId} [delete]
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

// AddReaction godoc
// @Summary Add reaction to message
// @Description Adds an emoji reaction to a message
// @Tags messages
// @Accept json
// @Security Bearer
// @Param messageId path string true "Message ID"
// @Param request body AddReactionRequest true "Reaction data"
// @Success 201 "Reaction added"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Message not found"
// @Router /chats/messages/{messageId}/reactions [post]
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

// RemoveReaction godoc
// @Summary Remove reaction from message
// @Description Removes an emoji reaction from a message
// @Tags messages
// @Security Bearer
// @Param messageId path string true "Message ID"
// @Param emoji path string true "Emoji to remove"
// @Success 204 "Reaction removed"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Message or reaction not found"
// @Router /chats/messages/{messageId}/reactions/{emoji} [delete]
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

// ListChatThreads godoc
// @Summary List chat threads
// @Description Returns paginated list of threads in a chat
// @Tags threads
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(20)
// @Success 200 {object} ThreadListResponse "Threads list"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/threads [get]
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

// CreateChatThread godoc
// @Summary Create a thread
// @Description Creates a new thread in a chat
// @Tags threads
// @Accept json
// @Produce json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param request body CreateThreadRequest true "Thread creation data"
// @Success 201 {object} ThreadResponse "Thread created"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/threads [post]
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

// GetThreadByID godoc
// @Summary Get thread by ID
// @Description Returns detailed information about a specific thread
// @Tags threads
// @Produce json
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Success 200 {object} ThreadResponse "Thread details"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId} [get]
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

// ArchiveThreadByID godoc
// @Summary Archive a thread
// @Description Archives a thread, making it read-only
// @Tags threads
// @Produce json
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Success 200 {object} ThreadResponse "Archived thread"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId}/archive [post]
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

// CreateThread godoc
// @Summary Create thread from message
// @Description Creates a reply thread from an existing message (backward compatibility)
// @Tags threads
// @Produce json
// @Security Bearer
// @Param messageId path string true "Message ID"
// @Success 201 {object} ThreadResponse "Thread created"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Message not found"
// @Router /chats/messages/{messageId}/thread [post]
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

// GetThreadMessages godoc
// @Summary List thread messages
// @Description Returns paginated list of messages in a thread
// @Tags threads
// @Produce json
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(50)
// @Success 200 {object} MessageListResponse "Messages list"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId}/messages [get]
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

// AddThreadParticipantHandler godoc
// @Summary Add participant to thread
// @Description Adds a new participant to a thread
// @Tags threads
// @Accept json
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Param request body map[string]string true "User ID to add"
// @Success 201 "Participant added"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId}/participants [post]
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

// RemoveThreadParticipantHandler godoc
// @Summary Remove participant from thread
// @Description Removes a participant from a thread
// @Tags threads
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Param userId path string true "User ID to remove"
// @Success 204 "Participant removed"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied"
// @Failure 404 {object} ErrorResponse "Thread or user not found"
// @Router /chats/threads/{threadId}/participants/{userId} [delete]
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

// ListThreadParticipantsHandler godoc
// @Summary List thread participants
// @Description Returns list of participants in a thread
// @Tags threads
// @Produce json
// @Security Bearer
// @Param threadId path string true "Thread ID"
// @Success 200 {object} map[string]interface{} "Participants list"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId}/participants [get]
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

// ListSubthreads godoc
// @Summary List subthreads
// @Description Returns paginated list of subthreads within a parent thread
// @Tags threads
// @Produce json
// @Security Bearer
// @Param threadId path string true "Parent Thread ID"
// @Param page query int false "Page number" default(1)
// @Param count query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Subthreads list"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Thread not found"
// @Router /chats/threads/{threadId}/subthreads [get]
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

// CreateSubthread godoc
// @Summary Create subthread
// @Description Creates a new subthread under a parent thread
// @Tags threads
// @Accept json
// @Produce json
// @Security Bearer
// @Param threadId path string true "Parent Thread ID"
// @Param request body map[string]string true "Subthread data"
// @Success 201 {object} ThreadResponse "Subthread created"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Parent thread not found"
// @Router /chats/threads/{threadId}/subthreads [post]
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

// AddToFavorites godoc
// @Summary Add chat to favorites
// @Description Adds a chat to user's favorites list
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 201 "Added to favorites"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/favorite [post]
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

// RemoveFromFavorites godoc
// @Summary Remove chat from favorites
// @Description Removes a chat from user's favorites list
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 204 "Removed from favorites"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/favorite [delete]
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

// ArchiveChat godoc
// @Summary Archive chat
// @Description Archives a chat for the current user
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 200 "Chat archived"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/archive [post]
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

// UnarchiveChat godoc
// @Summary Unarchive chat
// @Description Removes a chat from user's archive
// @Tags chats
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Success 200 "Chat unarchived"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/archive [delete]
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

// SendTypingIndicator godoc
// @Summary Send typing indicator
// @Description Sends typing status to notify other chat participants
// @Tags chats
// @Accept json
// @Security Bearer
// @Param chatId path string true "Chat ID"
// @Param request body map[string]bool true "Typing status"
// @Success 200 "Typing indicator sent"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "Chat not found"
// @Router /chats/{chatId}/typing [post]
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

// ForwardMessage godoc
// @Summary Forward a message
// @Description Forwards an existing message to another chat
// @Tags messages
// @Accept json
// @Produce json
// @Security Bearer
// @Param messageId path string true "Message ID to forward"
// @Param request body ForwardMessageRequest true "Forward destination"
// @Success 201 {object} MessageResponse "Forwarded message"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Access denied (guests cannot forward)"
// @Failure 404 {object} ErrorResponse "Message or target chat not found"
// @Router /chats/messages/{messageId}/forward [post]
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
