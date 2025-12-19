package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/services/voice/internal/model"
)

var (
	ErrCallNotFound = errors.New("call not found")
)

type CallRepository interface {
	CreateCall(ctx context.Context, call *model.Call) error
	GetCall(ctx context.Context, id uuid.UUID) (*model.Call, error)
	GetCallByFSUUID(ctx context.Context, fsUUID string) (*model.Call, error)
	UpdateCallStatus(ctx context.Context, id uuid.UUID, status model.CallStatus) error
	UpdateCallAnswered(ctx context.Context, id uuid.UUID, answeredAt time.Time) error
	UpdateCallEnded(ctx context.Context, id uuid.UUID, endedAt time.Time, duration int, endReason string) error
	GetCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Call, int, error)
	GetActiveCallForUser(ctx context.Context, userID uuid.UUID) (*model.Call, error)
}

type callRepository struct {
	pool *pgxpool.Pool
}

func NewCallRepository(pool *pgxpool.Pool) CallRepository {
	return &callRepository{pool: pool}
}

// CreateCall creates a new call record
func (r *callRepository) CreateCall(ctx context.Context, call *model.Call) error {
	query := `
		INSERT INTO con_test.calls (id, caller_id, callee_id, chat_id, conference_id, status, fs_call_uuid, started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	if call.ID == uuid.Nil {
		call.ID = uuid.New()
	}
	call.CreatedAt = now
	call.StartedAt = &now

	_, err := r.pool.Exec(ctx, query,
		call.ID, call.CallerID, call.CalleeID, call.ChatID, call.ConferenceID,
		call.Status, call.FSCallUUID, call.StartedAt, call.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create call: %w", err)
	}

	return nil
}

// GetCall retrieves a call by ID
func (r *callRepository) GetCall(ctx context.Context, id uuid.UUID) (*model.Call, error) {
	query := `
		SELECT c.id, c.caller_id, c.callee_id, c.chat_id, c.conference_id, c.status,
		       c.fs_call_uuid, c.duration, c.end_reason, c.started_at, c.answered_at,
		       c.ended_at, c.created_at,
		       caller.username AS caller_username, caller.display_name AS caller_display_name,
		       callee.username AS callee_username, callee.display_name AS callee_display_name
		FROM con_test.calls c
		LEFT JOIN con_test.users caller ON c.caller_id = caller.id
		LEFT JOIN con_test.users callee ON c.callee_id = callee.id
		WHERE c.id = $1
	`

	var call model.Call
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&call.ID, &call.CallerID, &call.CalleeID, &call.ChatID, &call.ConferenceID,
		&call.Status, &call.FSCallUUID, &call.Duration, &call.EndReason,
		&call.StartedAt, &call.AnsweredAt, &call.EndedAt, &call.CreatedAt,
		&call.CallerUsername, &call.CallerDisplayName,
		&call.CalleeUsername, &call.CalleeDisplayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCallNotFound
		}
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	return &call, nil
}

// GetCallByFSUUID retrieves a call by FreeSWITCH UUID
func (r *callRepository) GetCallByFSUUID(ctx context.Context, fsUUID string) (*model.Call, error) {
	query := `
		SELECT c.id, c.caller_id, c.callee_id, c.chat_id, c.conference_id, c.status,
		       c.fs_call_uuid, c.duration, c.end_reason, c.started_at, c.answered_at,
		       c.ended_at, c.created_at,
		       caller.username AS caller_username, caller.display_name AS caller_display_name,
		       callee.username AS callee_username, callee.display_name AS callee_display_name
		FROM con_test.calls c
		LEFT JOIN con_test.users caller ON c.caller_id = caller.id
		LEFT JOIN con_test.users callee ON c.callee_id = callee.id
		WHERE c.fs_call_uuid = $1
	`

	var call model.Call
	err := r.pool.QueryRow(ctx, query, fsUUID).Scan(
		&call.ID, &call.CallerID, &call.CalleeID, &call.ChatID, &call.ConferenceID,
		&call.Status, &call.FSCallUUID, &call.Duration, &call.EndReason,
		&call.StartedAt, &call.AnsweredAt, &call.EndedAt, &call.CreatedAt,
		&call.CallerUsername, &call.CallerDisplayName,
		&call.CalleeUsername, &call.CalleeDisplayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCallNotFound
		}
		return nil, fmt.Errorf("failed to get call by FS UUID: %w", err)
	}

	return &call, nil
}

// UpdateCallStatus updates call status
func (r *callRepository) UpdateCallStatus(ctx context.Context, id uuid.UUID, status model.CallStatus) error {
	query := `
		UPDATE con_test.calls SET status = $2 WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update call status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCallNotFound
	}

	return nil
}

// UpdateCallAnswered updates call when answered
func (r *callRepository) UpdateCallAnswered(ctx context.Context, id uuid.UUID, answeredAt time.Time) error {
	query := `
		UPDATE con_test.calls SET status = 'answered', answered_at = $2 WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, answeredAt)
	if err != nil {
		return fmt.Errorf("failed to update call answered: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCallNotFound
	}

	return nil
}

// UpdateCallEnded updates call when ended
func (r *callRepository) UpdateCallEnded(ctx context.Context, id uuid.UUID, endedAt time.Time, duration int, endReason string) error {
	query := `
		UPDATE con_test.calls SET status = 'ended', ended_at = $2, duration = $3, end_reason = $4 WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, endedAt, duration, endReason)
	if err != nil {
		return fmt.Errorf("failed to update call ended: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrCallNotFound
	}

	return nil
}

// GetCallHistory retrieves call history for a user
func (r *callRepository) GetCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Call, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM con_test.calls
		WHERE caller_id = $1 OR callee_id = $1
	`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count call history: %w", err)
	}

	query := `
		SELECT c.id, c.caller_id, c.callee_id, c.chat_id, c.conference_id, c.status,
		       c.fs_call_uuid, c.duration, c.end_reason, c.started_at, c.answered_at,
		       c.ended_at, c.created_at,
		       caller.username AS caller_username, caller.display_name AS caller_display_name,
		       callee.username AS callee_username, callee.display_name AS callee_display_name
		FROM con_test.calls c
		LEFT JOIN con_test.users caller ON c.caller_id = caller.id
		LEFT JOIN con_test.users callee ON c.callee_id = callee.id
		WHERE c.caller_id = $1 OR c.callee_id = $1
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get call history: %w", err)
	}
	defer rows.Close()

	calls := make([]*model.Call, 0)
	for rows.Next() {
		var call model.Call
		err := rows.Scan(
			&call.ID, &call.CallerID, &call.CalleeID, &call.ChatID, &call.ConferenceID,
			&call.Status, &call.FSCallUUID, &call.Duration, &call.EndReason,
			&call.StartedAt, &call.AnsweredAt, &call.EndedAt, &call.CreatedAt,
			&call.CallerUsername, &call.CallerDisplayName,
			&call.CalleeUsername, &call.CalleeDisplayName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan call: %w", err)
		}
		calls = append(calls, &call)
	}

	return calls, total, nil
}

// GetActiveCallForUser retrieves active call for a user (as caller or callee)
func (r *callRepository) GetActiveCallForUser(ctx context.Context, userID uuid.UUID) (*model.Call, error) {
	query := `
		SELECT c.id, c.caller_id, c.callee_id, c.chat_id, c.conference_id, c.status,
		       c.fs_call_uuid, c.duration, c.end_reason, c.started_at, c.answered_at,
		       c.ended_at, c.created_at,
		       caller.username AS caller_username, caller.display_name AS caller_display_name,
		       callee.username AS callee_username, callee.display_name AS callee_display_name
		FROM con_test.calls c
		LEFT JOIN con_test.users caller ON c.caller_id = caller.id
		LEFT JOIN con_test.users callee ON c.callee_id = callee.id
		WHERE (c.caller_id = $1 OR c.callee_id = $1)
		  AND c.status IN ('initiated', 'ringing', 'answered')
		ORDER BY c.created_at DESC
		LIMIT 1
	`

	var call model.Call
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&call.ID, &call.CallerID, &call.CalleeID, &call.ChatID, &call.ConferenceID,
		&call.Status, &call.FSCallUUID, &call.Duration, &call.EndReason,
		&call.StartedAt, &call.AnsweredAt, &call.EndedAt, &call.CreatedAt,
		&call.CallerUsername, &call.CallerDisplayName,
		&call.CalleeUsername, &call.CalleeDisplayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCallNotFound
		}
		return nil, fmt.Errorf("failed to get active call: %w", err)
	}

	return &call, nil
}
