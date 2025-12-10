package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/services/chat/internal/model"
)

var (
	ErrChatNotFound        = errors.New("chat not found")
	ErrMessageNotFound     = errors.New("message not found")
	ErrParticipantNotFound = errors.New("participant not found")
	ErrThreadNotFound      = errors.New("thread not found")
	ErrAlreadyExists       = errors.New("already exists")
	ErrAccessDenied        = errors.New("access denied")
)

type ChatRepository interface {
	// Chat operations
	CreateChat(ctx context.Context, chat *model.Chat) error
	GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error)
	ListChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error)
	UpdateChat(ctx context.Context, chat *model.Chat) error
	DeleteChat(ctx context.Context, id uuid.UUID) error
	SearchChats(ctx context.Context, userID uuid.UUID, query string, page, count int) ([]model.Chat, int, error)

	// Participant operations
	AddParticipant(ctx context.Context, participant *model.ChatParticipant) error
	GetParticipant(ctx context.Context, chatID, userID uuid.UUID) (*model.ChatParticipant, error)
	ListParticipants(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.ChatParticipant, int, error)
	GetParticipantIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
	UpdateParticipantRole(ctx context.Context, chatID, userID uuid.UUID, role model.ParticipantRole) error
	RemoveParticipant(ctx context.Context, chatID, userID uuid.UUID) error
	IsParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error)

	// Message operations
	CreateMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*model.Message, error)
	ListMessages(ctx context.Context, chatID uuid.UUID, page, count int, before, after *time.Time) ([]model.Message, int, error)
	GetMessagesSince(ctx context.Context, chatID uuid.UUID, afterSeqNum int64, limit int) ([]model.Message, error)
	UpdateMessage(ctx context.Context, message *model.Message) error
	DeleteMessage(ctx context.Context, id uuid.UUID) error
	GetThreadMessages(ctx context.Context, parentID uuid.UUID, page, count int) ([]model.Message, int, error)
	GetThreadCount(ctx context.Context, messageID uuid.UUID) (int, error)

	// File attachments
	GetAllFileLinkIDsForChat(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
	GetMessageAttachmentsBatch(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)

	// Reaction operations
	AddReaction(ctx context.Context, reaction *model.Reaction) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error
	ListReactions(ctx context.Context, messageID uuid.UUID) ([]model.Reaction, error)

	// Read status
	MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error
	GetReaders(ctx context.Context, messageID uuid.UUID) ([]uuid.UUID, error)
	GetUnreadCount(ctx context.Context, chatID, userID uuid.UUID) (int, error)

	// Favorites
	AddToFavorites(ctx context.Context, chatID, userID uuid.UUID) error
	RemoveFromFavorites(ctx context.Context, chatID, userID uuid.UUID) error
	IsFavorite(ctx context.Context, chatID, userID uuid.UUID) (bool, error)

	// Archive
	ArchiveChat(ctx context.Context, chatID, userID uuid.UUID) error
	UnarchiveChat(ctx context.Context, chatID, userID uuid.UUID) error
	ListArchivedChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error)

	// Thread operations
	CreateThread(ctx context.Context, thread *model.Thread) error
	GetThread(ctx context.Context, id uuid.UUID) (*model.Thread, error)
	GetThreadByParentMessage(ctx context.Context, parentMessageID uuid.UUID) (*model.Thread, error)
	GetSystemThread(ctx context.Context, chatID uuid.UUID) (*model.Thread, error)
	ListThreads(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.Thread, int, error)
	UpdateThread(ctx context.Context, thread *model.Thread) error
	ArchiveThread(ctx context.Context, threadID uuid.UUID) error

	// Thread participant operations
	AddThreadParticipant(ctx context.Context, participant *model.ThreadParticipant) error
	RemoveThreadParticipant(ctx context.Context, threadID, userID uuid.UUID) error
	ListThreadParticipants(ctx context.Context, threadID uuid.UUID) ([]model.ThreadParticipant, error)
	IsThreadParticipant(ctx context.Context, threadID, userID uuid.UUID) (bool, error)

	// Thread messages
	ListThreadMessages(ctx context.Context, threadID uuid.UUID, page, count int) ([]model.Message, int, error)

	// Thread access (cascading permissions)
	HasThreadAccess(ctx context.Context, threadID, userID uuid.UUID) (bool, error)
	GetThreadPermissionSource(ctx context.Context, threadID, userID uuid.UUID) (*model.PermissionSource, error)
	ListThreadsForUser(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Thread, int, error)
	ListSubthreads(ctx context.Context, parentThreadID uuid.UUID, userID uuid.UUID, page, count int) ([]model.Thread, int, error)
}

type chatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) ChatRepository {
	return &chatRepository{pool: pool}
}

// Chat operations

func (r *chatRepository) CreateChat(ctx context.Context, chat *model.Chat) error {
	query := `
		INSERT INTO con_test.chats (id, name, chat_type, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	chat.ID = uuid.New()
	now := time.Now()
	chat.CreatedAt = now
	chat.UpdatedAt = now

	_, err := r.pool.Exec(ctx, query,
		chat.ID, chat.Name, chat.ChatType, chat.CreatedBy, chat.CreatedAt, chat.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create chat: %w", err)
	}

	return nil
}

func (r *chatRepository) GetChat(ctx context.Context, id uuid.UUID) (*model.Chat, error) {
	query := `
		SELECT id, name, chat_type, created_by, created_at, updated_at
		FROM con_test.chats
		WHERE id = $1
	`

	var chat model.Chat
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&chat.ID, &chat.Name, &chat.ChatType, &chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChatNotFound
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	return &chat, nil
}

func (r *chatRepository) ListChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	countQuery := `
		SELECT COUNT(*) FROM con_test.chats c
		JOIN con_test.chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count chats: %w", err)
	}

	query := `
		SELECT c.id, c.name, c.chat_type, c.created_by, c.created_at, c.updated_at
		FROM con_test.chats c
		JOIN con_test.chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1
		ORDER BY c.updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list chats: %w", err)
	}
	defer rows.Close()

	var chats []model.Chat
	for rows.Next() {
		var chat model.Chat
		if err := rows.Scan(&chat.ID, &chat.Name, &chat.ChatType, &chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan chat: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, total, nil
}

func (r *chatRepository) UpdateChat(ctx context.Context, chat *model.Chat) error {
	query := `
		UPDATE con_test.chats SET name = $2, updated_at = $3 WHERE id = $1
	`
	chat.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query, chat.ID, chat.Name, chat.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update chat: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrChatNotFound
	}
	return nil
}

func (r *chatRepository) DeleteChat(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM con_test.chats WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrChatNotFound
	}
	return nil
}

func (r *chatRepository) SearchChats(ctx context.Context, userID uuid.UUID, query string, page, count int) ([]model.Chat, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count
	searchPattern := "%" + query + "%"

	countQuery := `
		SELECT COUNT(*) FROM con_test.chats c
		JOIN con_test.chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1 AND c.name ILIKE $2
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID, searchPattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count chats: %w", err)
	}

	sqlQuery := `
		SELECT c.id, c.name, c.chat_type, c.created_by, c.created_at, c.updated_at
		FROM con_test.chats c
		JOIN con_test.chat_participants cp ON c.id = cp.chat_id
		WHERE cp.user_id = $1 AND c.name ILIKE $2
		ORDER BY c.updated_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, sqlQuery, userID, searchPattern, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search chats: %w", err)
	}
	defer rows.Close()

	var chats []model.Chat
	for rows.Next() {
		var chat model.Chat
		if err := rows.Scan(&chat.ID, &chat.Name, &chat.ChatType, &chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan chat: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, total, nil
}

// Participant operations

func (r *chatRepository) AddParticipant(ctx context.Context, participant *model.ChatParticipant) error {
	query := `
		INSERT INTO con_test.chat_participants (id, chat_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`

	participant.ID = uuid.New()
	participant.JoinedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		participant.ID, participant.ChatID, participant.UserID, participant.Role, participant.JoinedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}
	return nil
}

func (r *chatRepository) GetParticipant(ctx context.Context, chatID, userID uuid.UUID) (*model.ChatParticipant, error) {
	query := `
		SELECT cp.id, cp.chat_id, cp.user_id, cp.role, cp.joined_at,
		       u.username, u.email, u.display_name, u.avatar_url
		FROM con_test.chat_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.chat_id = $1 AND cp.user_id = $2
	`

	var p model.ChatParticipant
	err := r.pool.QueryRow(ctx, query, chatID, userID).Scan(
		&p.ID, &p.ChatID, &p.UserID, &p.Role, &p.JoinedAt,
		&p.Username, &p.Email, &p.DisplayName, &p.AvatarURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}
	return &p, nil
}

func (r *chatRepository) ListParticipants(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.ChatParticipant, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.chat_participants WHERE chat_id = $1`, chatID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count participants: %w", err)
	}

	query := `
		SELECT cp.id, cp.chat_id, cp.user_id, cp.role, cp.joined_at,
		       u.username, u.email, u.display_name, u.avatar_url
		FROM con_test.chat_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.chat_id = $1
		ORDER BY cp.joined_at
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list participants: %w", err)
	}
	defer rows.Close()

	var participants []model.ChatParticipant
	for rows.Next() {
		var p model.ChatParticipant
		if err := rows.Scan(&p.ID, &p.ChatID, &p.UserID, &p.Role, &p.JoinedAt,
			&p.Username, &p.Email, &p.DisplayName, &p.AvatarURL); err != nil {
			return nil, 0, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	return participants, total, nil
}

func (r *chatRepository) GetParticipantIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT user_id FROM con_test.chat_participants WHERE chat_id = $1`
	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan participant ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *chatRepository) UpdateParticipantRole(ctx context.Context, chatID, userID uuid.UUID, role model.ParticipantRole) error {
	query := `UPDATE con_test.chat_participants SET role = $3 WHERE chat_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, chatID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to update participant role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *chatRepository) RemoveParticipant(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `DELETE FROM con_test.chat_participants WHERE chat_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *chatRepository) IsParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM con_test.chat_participants WHERE chat_id = $1 AND user_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, chatID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check participant: %w", err)
	}
	return exists, nil
}

// Message operations

func (r *chatRepository) CreateMessage(ctx context.Context, message *model.Message) error {
	// Start transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get next sequence number atomically
	var seqNum int64
	seqQuery := `SELECT con_test.get_next_seq_num($1)`
	err = tx.QueryRow(ctx, seqQuery, message.ChatID).Scan(&seqNum)
	if err != nil {
		return fmt.Errorf("failed to get next seq_num: %w", err)
	}

	query := `
		INSERT INTO con_test.messages (id, chat_id, parent_id, sender_id, content, sent_at, is_deleted, seq_num, forwarded_from_message_id, forwarded_from_chat_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	message.ID = uuid.New()
	message.SentAt = time.Now()
	message.IsDeleted = false
	message.SeqNum = seqNum

	_, err = tx.Exec(ctx, query,
		message.ID, message.ChatID, message.ParentID, message.SenderID, message.Content, message.SentAt, message.IsDeleted, message.SeqNum,
		message.ForwardedFromMessageID, message.ForwardedFromChatID,
	)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Save file attachments if any
	if len(message.FileLinkIDs) > 0 {
		attachQuery := `
			INSERT INTO con_test.message_file_attachments (id, message_id, file_link_id, sort_order)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (message_id, file_link_id) DO NOTHING
		`
		for i, linkID := range message.FileLinkIDs {
			_, err = tx.Exec(ctx, attachQuery, uuid.New(), message.ID, linkID, i)
			if err != nil {
				return fmt.Errorf("failed to create file attachment: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *chatRepository) GetMessage(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	query := `
		SELECT id, chat_id, parent_id, sender_id, content, sent_at, updated_at, is_deleted, seq_num,
		       forwarded_from_message_id, forwarded_from_chat_id
		FROM con_test.messages
		WHERE id = $1
	`

	var msg model.Message
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ChatID, &msg.ParentID, &msg.SenderID, &msg.Content, &msg.SentAt, &msg.UpdatedAt, &msg.IsDeleted, &msg.SeqNum,
		&msg.ForwardedFromMessageID, &msg.ForwardedFromChatID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Load file link IDs
	attachQuery := `
		SELECT file_link_id FROM con_test.message_file_attachments
		WHERE message_id = $1
		ORDER BY sort_order
	`
	rows, err := r.pool.Query(ctx, attachQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get file attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var linkID uuid.UUID
		if err := rows.Scan(&linkID); err != nil {
			return nil, fmt.Errorf("failed to scan file link ID: %w", err)
		}
		msg.FileLinkIDs = append(msg.FileLinkIDs, linkID)
	}

	return &msg, nil
}

func (r *chatRepository) ListMessages(ctx context.Context, chatID uuid.UUID, page, count int, before, after *time.Time) ([]model.Message, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 50
	}
	offset := (page - 1) * count

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.messages WHERE chat_id = $1 AND parent_id IS NULL`, chatID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	query := `
		SELECT m.id, m.chat_id, m.parent_id, m.sender_id, m.content, m.sent_at, m.updated_at, m.is_deleted, m.seq_num,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.messages m
		LEFT JOIN con_test.users u ON m.sender_id = u.id
		WHERE m.chat_id = $1 AND m.parent_id IS NULL
		ORDER BY m.sent_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.ParentID, &msg.SenderID, &msg.Content, &msg.SentAt, &msg.UpdatedAt, &msg.IsDeleted, &msg.SeqNum,
			&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, total, nil
}

// GetMessagesSince returns messages with seq_num > afterSeqNum, ordered by seq_num ASC
// Used for syncing missed messages after reconnect
func (r *chatRepository) GetMessagesSince(ctx context.Context, chatID uuid.UUID, afterSeqNum int64, limit int) ([]model.Message, error) {
	if limit < 1 || limit > 1000 {
		limit = 100
	}

	query := `
		SELECT m.id, m.chat_id, m.parent_id, m.sender_id, m.content, m.sent_at, m.updated_at, m.is_deleted, m.seq_num,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.messages m
		LEFT JOIN con_test.users u ON m.sender_id = u.id
		WHERE m.chat_id = $1 AND m.seq_num > $2
		ORDER BY m.seq_num ASC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, afterSeqNum, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages since seq_num: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.ParentID, &msg.SenderID, &msg.Content, &msg.SentAt, &msg.UpdatedAt, &msg.IsDeleted, &msg.SeqNum,
			&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (r *chatRepository) UpdateMessage(ctx context.Context, message *model.Message) error {
	query := `UPDATE con_test.messages SET content = $2, updated_at = $3 WHERE id = $1`
	now := time.Now()
	message.UpdatedAt = &now

	result, err := r.pool.Exec(ctx, query, message.ID, message.Content, message.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *chatRepository) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE con_test.messages SET is_deleted = true, updated_at = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMessageNotFound
	}
	return nil
}

func (r *chatRepository) GetThreadMessages(ctx context.Context, parentID uuid.UUID, page, count int) ([]model.Message, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 50
	}
	offset := (page - 1) * count

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.messages WHERE parent_id = $1`, parentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count thread messages: %w", err)
	}

	query := `
		SELECT id, chat_id, parent_id, sender_id, content, sent_at, updated_at, is_deleted, seq_num
		FROM con_test.messages
		WHERE parent_id = $1
		ORDER BY sent_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, parentID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list thread messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(&msg.ID, &msg.ChatID, &msg.ParentID, &msg.SenderID, &msg.Content, &msg.SentAt, &msg.UpdatedAt, &msg.IsDeleted, &msg.SeqNum); err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, total, nil
}

func (r *chatRepository) GetThreadCount(ctx context.Context, messageID uuid.UUID) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.messages WHERE parent_id = $1`, messageID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count thread: %w", err)
	}
	return count, nil
}

// Reaction operations

func (r *chatRepository) AddReaction(ctx context.Context, reaction *model.Reaction) error {
	query := `
		INSERT INTO con_test.message_reactions (id, message_id, user_id, reaction, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id, user_id, reaction) DO NOTHING
	`

	reaction.ID = uuid.New()
	reaction.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query, reaction.ID, reaction.MessageID, reaction.UserID, reaction.Reaction, reaction.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	return nil
}

func (r *chatRepository) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, reaction string) error {
	query := `DELETE FROM con_test.message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction = $3`
	_, err := r.pool.Exec(ctx, query, messageID, userID, reaction)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}
	return nil
}

func (r *chatRepository) ListReactions(ctx context.Context, messageID uuid.UUID) ([]model.Reaction, error) {
	query := `SELECT id, message_id, user_id, reaction, created_at FROM con_test.message_reactions WHERE message_id = $1`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to list reactions: %w", err)
	}
	defer rows.Close()

	var reactions []model.Reaction
	for rows.Next() {
		var r model.Reaction
		if err := rows.Scan(&r.ID, &r.MessageID, &r.UserID, &r.Reaction, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan reaction: %w", err)
		}
		reactions = append(reactions, r)
	}
	return reactions, nil
}

// Read status

func (r *chatRepository) MarkAsRead(ctx context.Context, messageID, userID uuid.UUID) error {
	query := `
		INSERT INTO con_test.message_readers (id, message_id, user_id, read_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (message_id, user_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, uuid.New(), messageID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}
	return nil
}

func (r *chatRepository) GetReaders(ctx context.Context, messageID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT user_id FROM con_test.message_readers WHERE message_id = $1`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get readers: %w", err)
	}
	defer rows.Close()

	var readers []uuid.UUID
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan reader: %w", err)
		}
		readers = append(readers, userID)
	}
	return readers, nil
}

func (r *chatRepository) GetUnreadCount(ctx context.Context, chatID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM con_test.messages m
		WHERE m.chat_id = $1 AND m.sender_id != $2
		AND NOT EXISTS (SELECT 1 FROM con_test.message_readers mr WHERE mr.message_id = m.id AND mr.user_id = $2)
	`
	var count int
	if err := r.pool.QueryRow(ctx, query, chatID, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count unread: %w", err)
	}
	return count, nil
}

// Favorites

func (r *chatRepository) AddToFavorites(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `
		INSERT INTO con_test.chat_favorites (id, chat_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, uuid.New(), chatID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to add to favorites: %w", err)
	}
	return nil
}

func (r *chatRepository) RemoveFromFavorites(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `DELETE FROM con_test.chat_favorites WHERE chat_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove from favorites: %w", err)
	}
	return nil
}

func (r *chatRepository) IsFavorite(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM con_test.chat_favorites WHERE chat_id = $1 AND user_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, chatID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check favorite: %w", err)
	}
	return exists, nil
}

// Archive

func (r *chatRepository) ArchiveChat(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `
		INSERT INTO con_test.archived_chats (id, chat_id, user_id, archived_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chat_id, user_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, query, uuid.New(), chatID, userID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to archive chat: %w", err)
	}
	return nil
}

func (r *chatRepository) UnarchiveChat(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `DELETE FROM con_test.archived_chats WHERE chat_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to unarchive chat: %w", err)
	}
	return nil
}

func (r *chatRepository) ListArchivedChats(ctx context.Context, userID uuid.UUID, page, count int) ([]model.Chat, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	var total int
	countQuery := `SELECT COUNT(*) FROM con_test.archived_chats WHERE user_id = $1`
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count archived chats: %w", err)
	}

	query := `
		SELECT c.id, c.name, c.chat_type, c.created_by, c.created_at, c.updated_at
		FROM con_test.chats c
		JOIN con_test.archived_chats ac ON c.id = ac.chat_id
		WHERE ac.user_id = $1
		ORDER BY ac.archived_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list archived chats: %w", err)
	}
	defer rows.Close()

	var chats []model.Chat
	for rows.Next() {
		var chat model.Chat
		if err := rows.Scan(&chat.ID, &chat.Name, &chat.ChatType, &chat.CreatedBy, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan chat: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, total, nil
}

// File attachments

func (r *chatRepository) GetAllFileLinkIDsForChat(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT DISTINCT mfa.file_link_id
		FROM con_test.message_file_attachments mfa
		JOIN con_test.messages m ON m.id = mfa.message_id
		WHERE m.chat_id = $1 AND m.is_deleted = false
	`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file link IDs for chat: %w", err)
	}
	defer rows.Close()

	var linkIDs []uuid.UUID
	for rows.Next() {
		var linkID uuid.UUID
		if err := rows.Scan(&linkID); err != nil {
			return nil, fmt.Errorf("failed to scan file link ID: %w", err)
		}
		linkIDs = append(linkIDs, linkID)
	}

	return linkIDs, nil
}

func (r *chatRepository) GetMessageAttachmentsBatch(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	if len(messageIDs) == 0 {
		return make(map[uuid.UUID][]uuid.UUID), nil
	}

	query := `
		SELECT message_id, file_link_id
		FROM con_test.message_file_attachments
		WHERE message_id = ANY($1)
		ORDER BY message_id, sort_order
	`

	rows, err := r.pool.Query(ctx, query, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get message attachments: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]uuid.UUID)
	for rows.Next() {
		var messageID, linkID uuid.UUID
		if err := rows.Scan(&messageID, &linkID); err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		result[messageID] = append(result[messageID], linkID)
	}

	return result, nil
}

// Thread operations

func (r *chatRepository) CreateThread(ctx context.Context, thread *model.Thread) error {
	query := `
		INSERT INTO con_test.threads (id, chat_id, parent_message_id, parent_thread_id, thread_type, title, message_count, last_message_at, created_by, created_at, updated_at, is_archived, restricted_participants)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	thread.ID = uuid.New()
	now := time.Now()
	thread.CreatedAt = now
	thread.UpdatedAt = now
	// Depth is set by database trigger based on parent_thread_id

	_, err := r.pool.Exec(ctx, query,
		thread.ID, thread.ChatID, thread.ParentMessageID, thread.ParentThreadID, thread.ThreadType, thread.Title,
		thread.MessageCount, thread.LastMessageAt, thread.CreatedBy, thread.CreatedAt,
		thread.UpdatedAt, thread.IsArchived, thread.RestrictedParticipants,
	)
	if err != nil {
		return fmt.Errorf("failed to create thread: %w", err)
	}

	return nil
}

func (r *chatRepository) GetThread(ctx context.Context, id uuid.UUID) (*model.Thread, error) {
	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE id = $1
	`

	var thread model.Thread
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&thread.ID, &thread.ChatID, &thread.ParentMessageID, &thread.ParentThreadID, &thread.Depth,
		&thread.ThreadType, &thread.Title, &thread.MessageCount, &thread.LastMessageAt, &thread.CreatedBy,
		&thread.CreatedAt, &thread.UpdatedAt, &thread.IsArchived, &thread.RestrictedParticipants,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	return &thread, nil
}

func (r *chatRepository) GetThreadByParentMessage(ctx context.Context, parentMessageID uuid.UUID) (*model.Thread, error) {
	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE parent_message_id = $1
	`

	var thread model.Thread
	err := r.pool.QueryRow(ctx, query, parentMessageID).Scan(
		&thread.ID, &thread.ChatID, &thread.ParentMessageID, &thread.ParentThreadID, &thread.Depth,
		&thread.ThreadType, &thread.Title, &thread.MessageCount, &thread.LastMessageAt, &thread.CreatedBy,
		&thread.CreatedAt, &thread.UpdatedAt, &thread.IsArchived, &thread.RestrictedParticipants,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to get thread by parent message: %w", err)
	}

	return &thread, nil
}

func (r *chatRepository) GetSystemThread(ctx context.Context, chatID uuid.UUID) (*model.Thread, error) {
	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE chat_id = $1 AND thread_type = 'system'
		LIMIT 1
	`

	var thread model.Thread
	err := r.pool.QueryRow(ctx, query, chatID).Scan(
		&thread.ID, &thread.ChatID, &thread.ParentMessageID, &thread.ParentThreadID, &thread.Depth,
		&thread.ThreadType, &thread.Title, &thread.MessageCount, &thread.LastMessageAt, &thread.CreatedBy,
		&thread.CreatedAt, &thread.UpdatedAt, &thread.IsArchived, &thread.RestrictedParticipants,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, fmt.Errorf("failed to get system thread: %w", err)
	}

	return &thread, nil
}

func (r *chatRepository) ListThreads(ctx context.Context, chatID uuid.UUID, page, count int) ([]model.Thread, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.threads WHERE chat_id = $1 AND is_archived = false AND parent_thread_id IS NULL`, chatID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count threads: %w", err)
	}

	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE chat_id = $1 AND is_archived = false AND parent_thread_id IS NULL
		ORDER BY last_message_at DESC NULLS LAST, created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threads []model.Thread
	for rows.Next() {
		var t model.Thread
		if err := rows.Scan(
			&t.ID, &t.ChatID, &t.ParentMessageID, &t.ParentThreadID, &t.Depth,
			&t.ThreadType, &t.Title, &t.MessageCount, &t.LastMessageAt, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.IsArchived, &t.RestrictedParticipants,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan thread: %w", err)
		}
		threads = append(threads, t)
	}

	return threads, total, nil
}

func (r *chatRepository) UpdateThread(ctx context.Context, thread *model.Thread) error {
	query := `
		UPDATE con_test.threads
		SET title = $2, is_archived = $3, restricted_participants = $4, updated_at = $5
		WHERE id = $1
	`
	thread.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query, thread.ID, thread.Title, thread.IsArchived, thread.RestrictedParticipants, thread.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update thread: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrThreadNotFound
	}
	return nil
}

func (r *chatRepository) ArchiveThread(ctx context.Context, threadID uuid.UUID) error {
	query := `UPDATE con_test.threads SET is_archived = true, updated_at = $2 WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, threadID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to archive thread: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrThreadNotFound
	}
	return nil
}

// Thread participant operations

func (r *chatRepository) AddThreadParticipant(ctx context.Context, participant *model.ThreadParticipant) error {
	query := `
		INSERT INTO con_test.thread_participants (id, thread_id, user_id, added_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (thread_id, user_id) DO NOTHING
	`

	participant.ID = uuid.New()
	participant.AddedAt = time.Now()

	_, err := r.pool.Exec(ctx, query, participant.ID, participant.ThreadID, participant.UserID, participant.AddedAt)
	if err != nil {
		return fmt.Errorf("failed to add thread participant: %w", err)
	}
	return nil
}

func (r *chatRepository) RemoveThreadParticipant(ctx context.Context, threadID, userID uuid.UUID) error {
	query := `DELETE FROM con_test.thread_participants WHERE thread_id = $1 AND user_id = $2`
	result, err := r.pool.Exec(ctx, query, threadID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove thread participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (r *chatRepository) ListThreadParticipants(ctx context.Context, threadID uuid.UUID) ([]model.ThreadParticipant, error) {
	query := `
		SELECT id, thread_id, user_id, added_at
		FROM con_test.thread_participants
		WHERE thread_id = $1
		ORDER BY added_at
	`

	rows, err := r.pool.Query(ctx, query, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to list thread participants: %w", err)
	}
	defer rows.Close()

	var participants []model.ThreadParticipant
	for rows.Next() {
		var p model.ThreadParticipant
		if err := rows.Scan(&p.ID, &p.ThreadID, &p.UserID, &p.AddedAt); err != nil {
			return nil, fmt.Errorf("failed to scan thread participant: %w", err)
		}
		participants = append(participants, p)
	}
	return participants, nil
}

func (r *chatRepository) IsThreadParticipant(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM con_test.thread_participants WHERE thread_id = $1 AND user_id = $2)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, threadID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check thread participant: %w", err)
	}
	return exists, nil
}

// Thread messages

func (r *chatRepository) ListThreadMessages(ctx context.Context, threadID uuid.UUID, page, count int) ([]model.Message, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 50
	}
	offset := (page - 1) * count

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM con_test.messages WHERE thread_id = $1`, threadID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count thread messages: %w", err)
	}

	query := `
		SELECT m.id, m.chat_id, m.parent_id, m.thread_id, m.sender_id, m.content, m.sent_at, m.updated_at, m.is_deleted, m.is_system, m.seq_num,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.messages m
		LEFT JOIN con_test.users u ON m.sender_id = u.id
		WHERE m.thread_id = $1
		ORDER BY m.sent_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, threadID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list thread messages: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var msg model.Message
		if err := rows.Scan(
			&msg.ID, &msg.ChatID, &msg.ParentID, &msg.ThreadID, &msg.SenderID, &msg.Content,
			&msg.SentAt, &msg.UpdatedAt, &msg.IsDeleted, &msg.IsSystem, &msg.SeqNum,
			&msg.SenderUsername, &msg.SenderDisplayName, &msg.SenderAvatarURL,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan thread message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, total, nil
}

// HasThreadAccess checks if user has access to a thread using cascading permission check
// Uses PostgreSQL function check_thread_access for efficient recursive check
func (r *chatRepository) HasThreadAccess(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	query := `SELECT con_test.check_thread_access($1, $2)`
	var hasAccess bool
	if err := r.pool.QueryRow(ctx, query, threadID, userID).Scan(&hasAccess); err != nil {
		return false, fmt.Errorf("failed to check thread access: %w", err)
	}
	return hasAccess, nil
}

// GetThreadPermissionSource returns where user's permission to access thread comes from
func (r *chatRepository) GetThreadPermissionSource(ctx context.Context, threadID, userID uuid.UUID) (*model.PermissionSource, error) {
	query := `SELECT source, source_id FROM con_test.get_thread_permission_source($1, $2)`
	var source model.PermissionSource
	err := r.pool.QueryRow(ctx, query, threadID, userID).Scan(&source.Source, &source.SourceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccessDenied
		}
		return nil, fmt.Errorf("failed to get thread permission source: %w", err)
	}
	return &source, nil
}

// ListThreadsForUser lists threads in a chat filtered by user access
func (r *chatRepository) ListThreadsForUser(ctx context.Context, chatID, userID uuid.UUID, page, count int) ([]model.Thread, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	// Count only accessible threads (top-level, not subthreads)
	countQuery := `
		SELECT COUNT(*) FROM con_test.threads
		WHERE chat_id = $1 AND is_archived = false AND parent_thread_id IS NULL
		AND con_test.check_thread_access(id, $2)
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, chatID, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count threads: %w", err)
	}

	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE chat_id = $1 AND is_archived = false AND parent_thread_id IS NULL
		AND con_test.check_thread_access(id, $2)
		ORDER BY last_message_at DESC NULLS LAST, created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, chatID, userID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	var threads []model.Thread
	for rows.Next() {
		var t model.Thread
		if err := rows.Scan(
			&t.ID, &t.ChatID, &t.ParentMessageID, &t.ParentThreadID, &t.Depth,
			&t.ThreadType, &t.Title, &t.MessageCount, &t.LastMessageAt, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.IsArchived, &t.RestrictedParticipants,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan thread: %w", err)
		}
		threads = append(threads, t)
	}

	return threads, total, nil
}

// ListSubthreads lists subthreads of a parent thread filtered by user access
func (r *chatRepository) ListSubthreads(ctx context.Context, parentThreadID uuid.UUID, userID uuid.UUID, page, count int) ([]model.Thread, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}
	offset := (page - 1) * count

	// Count only accessible subthreads
	countQuery := `
		SELECT COUNT(*) FROM con_test.threads
		WHERE parent_thread_id = $1 AND is_archived = false
		AND con_test.check_thread_access(id, $2)
	`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, parentThreadID, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count subthreads: %w", err)
	}

	query := `
		SELECT id, chat_id, parent_message_id, parent_thread_id, depth, thread_type, title, message_count, last_message_at,
		       created_by, created_at, updated_at, is_archived, restricted_participants
		FROM con_test.threads
		WHERE parent_thread_id = $1 AND is_archived = false
		AND con_test.check_thread_access(id, $2)
		ORDER BY last_message_at DESC NULLS LAST, created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(ctx, query, parentThreadID, userID, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subthreads: %w", err)
	}
	defer rows.Close()

	var threads []model.Thread
	for rows.Next() {
		var t model.Thread
		if err := rows.Scan(
			&t.ID, &t.ChatID, &t.ParentMessageID, &t.ParentThreadID, &t.Depth,
			&t.ThreadType, &t.Title, &t.MessageCount, &t.LastMessageAt, &t.CreatedBy,
			&t.CreatedAt, &t.UpdatedAt, &t.IsArchived, &t.RestrictedParticipants,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan subthread: %w", err)
		}
		threads = append(threads, t)
	}

	return threads, total, nil
}
