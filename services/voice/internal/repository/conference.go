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
	ErrConferenceNotFound   = errors.New("conference not found")
	ErrParticipantNotFound  = errors.New("participant not found")
	ErrAlreadyParticipant   = errors.New("user is already a participant")
	ErrConferenceEnded      = errors.New("conference has ended")
	ErrConferenceFull       = errors.New("conference is full")
)

type ConferenceRepository interface {
	// Conference operations
	CreateConference(ctx context.Context, conf *model.Conference) error
	GetConference(ctx context.Context, id uuid.UUID) (*model.Conference, error)
	GetConferenceByFSName(ctx context.Context, fsName string) (*model.Conference, error)
	GetConferenceByChatID(ctx context.Context, chatID uuid.UUID) (*model.Conference, error)
	ListConferences(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*model.Conference, int, error)
	ListAllActiveConferences(ctx context.Context) ([]*model.Conference, error)
	UpdateConferenceStatus(ctx context.Context, id uuid.UUID, status model.ConferenceStatus, endedAt *time.Time) error
	SetRecordingPath(ctx context.Context, id uuid.UUID, path string) error

	// Participant operations
	AddParticipant(ctx context.Context, p *model.Participant) error
	GetParticipant(ctx context.Context, confID, userID uuid.UUID) (*model.Participant, error)
	GetParticipantByFSMemberID(ctx context.Context, confID uuid.UUID, fsMemberID string) (*model.Participant, error)
	GetParticipantByChannelUUID(ctx context.Context, channelUUID string) (*model.Participant, error)
	ListParticipants(ctx context.Context, confID uuid.UUID) ([]*model.Participant, error)
	UpdateParticipantStatus(ctx context.Context, id uuid.UUID, status model.ParticipantStatus, leftAt *time.Time) error
	UpdateParticipantMute(ctx context.Context, id uuid.UUID, muted bool) error
	UpdateParticipantSpeaking(ctx context.Context, id uuid.UUID, speaking bool) error
	SetParticipantFSMemberID(ctx context.Context, id uuid.UUID, fsMemberID string) error
	SetParticipantChannelUUID(ctx context.Context, id uuid.UUID, channelUUID string) error
	GetActiveParticipantCount(ctx context.Context, confID uuid.UUID) (int, error)
	CleanupStaleConnectingParticipants(ctx context.Context, timeout time.Duration) (int, error)

	// Scheduled events operations
	CreateScheduledConference(ctx context.Context, conf *model.Conference, recurrence *model.RecurrenceRule) error
	ListScheduledConferences(ctx context.Context, userID uuid.UUID, upcomingOnly bool, limit, offset int) ([]*model.Conference, int, error)
	GetChatConferences(ctx context.Context, chatID uuid.UUID, upcomingOnly bool) ([]*model.Conference, error)
	UpdateParticipantRSVP(ctx context.Context, participantID uuid.UUID, status model.RSVPStatus) error
	UpdateParticipantRole(ctx context.Context, participantID uuid.UUID, role model.ConferenceRole) error
	AddParticipantWithRole(ctx context.Context, p *model.Participant) error
	RemoveParticipant(ctx context.Context, confID, userID uuid.UUID) error
	GetConferenceWithParticipants(ctx context.Context, id uuid.UUID) (*model.Conference, error)

	// Recurrence operations
	CreateRecurrenceRule(ctx context.Context, rule *model.RecurrenceRule) error
	GetRecurrenceRule(ctx context.Context, conferenceID uuid.UUID) (*model.RecurrenceRule, error)

	// Reminder operations
	CreateReminder(ctx context.Context, reminder *model.ConferenceReminder) error
	GetPendingReminders(ctx context.Context, now time.Time) ([]*model.ConferenceReminder, error)
	MarkReminderSent(ctx context.Context, id uuid.UUID) error

	// Cleanup operations
	CleanupStaleConferences(ctx context.Context, maxAge time.Duration) (int, error)

	// History operations
	LogModeratorAction(ctx context.Context, action *model.ModeratorAction) error
	ListModeratorActions(ctx context.Context, conferenceID uuid.UUID) ([]model.ModeratorAction, error)
	ListConferenceHistory(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.ConferenceHistory, int, error)
	GetConferenceHistory(ctx context.Context, conferenceID uuid.UUID) (*model.ConferenceHistory, error)
	GetAllParticipantSessions(ctx context.Context, conferenceID uuid.UUID) ([]model.ParticipantHistory, error)
	GetConferenceMessages(ctx context.Context, confID, chatID uuid.UUID, startedAt, endedAt *time.Time) ([]*model.ConferenceMessage, error)
	SetConferenceThread(ctx context.Context, conferenceID, threadID uuid.UUID) error
	GetConferenceThread(ctx context.Context, conferenceID uuid.UUID) (*uuid.UUID, error)
}

type conferenceRepository struct {
	pool *pgxpool.Pool
}

func NewConferenceRepository(pool *pgxpool.Pool) ConferenceRepository {
	return &conferenceRepository{pool: pool}
}

// CreateConference creates a new conference
func (r *conferenceRepository) CreateConference(ctx context.Context, conf *model.Conference) error {
	query := `
		INSERT INTO con_test.conferences (id, name, chat_id, freeswitch_name, created_by, status, max_members, is_private, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now()
	if conf.ID == uuid.Nil {
		conf.ID = uuid.New()
	}
	conf.CreatedAt = now
	conf.UpdatedAt = now
	conf.StartedAt = &now

	_, err := r.pool.Exec(ctx, query,
		conf.ID, conf.Name, conf.ChatID, conf.FreeSwitchName, conf.CreatedBy,
		conf.Status, conf.MaxMembers, conf.IsPrivate, conf.StartedAt,
		conf.CreatedAt, conf.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create conference: %w", err)
	}

	return nil
}

// GetConference retrieves a conference by ID
func (r *conferenceRepository) GetConference(ctx context.Context, id uuid.UUID) (*model.Conference, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp
		        WHERE cp.conference_id = c.id AND cp.status = 'joined') AS participant_count
		FROM con_test.conferences c
		WHERE c.id = $1
	`

	var conf model.Conference
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
		&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
		&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
		&conf.ParticipantCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConferenceNotFound
		}
		return nil, fmt.Errorf("failed to get conference: %w", err)
	}

	return &conf, nil
}

// GetConferenceByFSName retrieves a conference by FreeSWITCH name
func (r *conferenceRepository) GetConferenceByFSName(ctx context.Context, fsName string) (*model.Conference, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp
		        WHERE cp.conference_id = c.id AND cp.status = 'joined') AS participant_count
		FROM con_test.conferences c
		WHERE c.freeswitch_name = $1
	`

	var conf model.Conference
	err := r.pool.QueryRow(ctx, query, fsName).Scan(
		&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
		&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
		&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
		&conf.ParticipantCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConferenceNotFound
		}
		return nil, fmt.Errorf("failed to get conference by FS name: %w", err)
	}

	return &conf, nil
}

// GetConferenceByChatID retrieves an active conference by chat ID
func (r *conferenceRepository) GetConferenceByChatID(ctx context.Context, chatID uuid.UUID) (*model.Conference, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp
		        WHERE cp.conference_id = c.id AND cp.status = 'joined') AS participant_count
		FROM con_test.conferences c
		WHERE c.chat_id = $1 AND c.status = 'active'
		ORDER BY c.created_at DESC
		LIMIT 1
	`

	var conf model.Conference
	err := r.pool.QueryRow(ctx, query, chatID).Scan(
		&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
		&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
		&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
		&conf.ParticipantCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConferenceNotFound
		}
		return nil, fmt.Errorf("failed to get conference by chat ID: %w", err)
	}

	return &conf, nil
}

// ListConferences lists conferences for a user
func (r *conferenceRepository) ListConferences(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*model.Conference, int, error) {
	baseQuery := `
		SELECT DISTINCT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp2
		        WHERE cp2.conference_id = c.id AND cp2.status = 'joined') AS participant_count
		FROM con_test.conferences c
		LEFT JOIN con_test.conference_participants cp ON c.id = cp.conference_id
		WHERE (c.created_by = $1 OR cp.user_id = $1)
	`

	countQuery := `
		SELECT COUNT(DISTINCT c.id)
		FROM con_test.conferences c
		LEFT JOIN con_test.conference_participants cp ON c.id = cp.conference_id
		WHERE (c.created_by = $1 OR cp.user_id = $1)
	`

	if activeOnly {
		baseQuery += " AND c.status = 'active'"
		countQuery += " AND c.status = 'active'"
	}

	baseQuery += " ORDER BY c.created_at DESC LIMIT $2 OFFSET $3"

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count conferences: %w", err)
	}

	// Get conferences
	rows, err := r.pool.Query(ctx, baseQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conferences: %w", err)
	}
	defer rows.Close()

	conferences := make([]*model.Conference, 0)
	for rows.Next() {
		var conf model.Conference
		err := rows.Scan(
			&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
			&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
			&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
			&conf.ParticipantCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan conference: %w", err)
		}
		conferences = append(conferences, &conf)
	}

	return conferences, total, nil
}

// ListAllActiveConferences returns all active conferences with chat_id (for UI indicators)
func (r *conferenceRepository) ListAllActiveConferences(ctx context.Context) ([]*model.Conference, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp2
		        WHERE cp2.conference_id = c.id AND cp2.status = 'joined') AS participant_count
		FROM con_test.conferences c
		WHERE c.status = 'active' AND c.chat_id IS NOT NULL
		ORDER BY c.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active conferences: %w", err)
	}
	defer rows.Close()

	conferences := make([]*model.Conference, 0)
	for rows.Next() {
		var conf model.Conference
		err := rows.Scan(
			&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
			&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
			&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
			&conf.ParticipantCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conference: %w", err)
		}
		conferences = append(conferences, &conf)
	}

	return conferences, nil
}

// UpdateConferenceStatus updates conference status
func (r *conferenceRepository) UpdateConferenceStatus(ctx context.Context, id uuid.UUID, status model.ConferenceStatus, endedAt *time.Time) error {
	query := `
		UPDATE con_test.conferences
		SET status = $2, ended_at = $3, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, status, endedAt)
	if err != nil {
		return fmt.Errorf("failed to update conference status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrConferenceNotFound
	}

	return nil
}

// SetRecordingPath sets the recording path for a conference
func (r *conferenceRepository) SetRecordingPath(ctx context.Context, id uuid.UUID, path string) error {
	query := `
		UPDATE con_test.conferences
		SET recording_path = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, path)
	if err != nil {
		return fmt.Errorf("failed to set recording path: %w", err)
	}

	return nil
}

// AddParticipant adds a participant to a conference
func (r *conferenceRepository) AddParticipant(ctx context.Context, p *model.Participant) error {
	query := `
		INSERT INTO con_test.conference_participants (id, conference_id, user_id, status, is_muted, is_deaf, joined_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = now
	if p.Status == model.ParticipantStatusJoined && p.JoinedAt == nil {
		p.JoinedAt = &now
	}

	_, err := r.pool.Exec(ctx, query,
		p.ID, p.ConferenceID, p.UserID, p.Status, p.IsMuted, p.IsDeaf, p.JoinedAt, p.CreatedAt)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() != "" && (err.Error() == "UNIQUE constraint" || err.Error() != "") {
			// Try to check if it's a duplicate
			existing, _ := r.GetParticipant(ctx, p.ConferenceID, p.UserID)
			if existing != nil {
				return ErrAlreadyParticipant
			}
		}
		return fmt.Errorf("failed to add participant: %w", err)
	}

	return nil
}

// GetParticipant retrieves a participant by conference and user ID
func (r *conferenceRepository) GetParticipant(ctx context.Context, confID, userID uuid.UUID) (*model.Participant, error) {
	query := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.conference_id = $1 AND cp.user_id = $2 AND cp.left_at IS NULL
	`

	var p model.Participant
	err := r.pool.QueryRow(ctx, query, confID, userID).Scan(
		&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
		&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
		&p.Role, &p.RSVPStatus, &p.RSVPAt,
		&p.Username, &p.DisplayName, &p.AvatarURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}

	return &p, nil
}

// GetParticipantByFSMemberID retrieves a participant by FreeSWITCH member ID
func (r *conferenceRepository) GetParticipantByFSMemberID(ctx context.Context, confID uuid.UUID, fsMemberID string) (*model.Participant, error) {
	query := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.conference_id = $1 AND cp.fs_member_id = $2 AND cp.left_at IS NULL
	`

	var p model.Participant
	err := r.pool.QueryRow(ctx, query, confID, fsMemberID).Scan(
		&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
		&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
		&p.Role, &p.RSVPStatus, &p.RSVPAt,
		&p.Username, &p.DisplayName, &p.AvatarURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("failed to get participant by FS member ID: %w", err)
	}

	return &p, nil
}

// ListParticipants lists all active participants in a conference
func (r *conferenceRepository) ListParticipants(ctx context.Context, confID uuid.UUID) ([]*model.Participant, error) {
	query := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.conference_id = $1 AND cp.status = 'joined'
		ORDER BY cp.joined_at
	`

	rows, err := r.pool.Query(ctx, query, confID)
	if err != nil {
		return nil, fmt.Errorf("failed to list participants: %w", err)
	}
	defer rows.Close()

	participants := make([]*model.Participant, 0)
	for rows.Next() {
		var p model.Participant
		err := rows.Scan(
			&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
			&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
			&p.Role, &p.RSVPStatus, &p.RSVPAt,
			&p.Username, &p.DisplayName, &p.AvatarURL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, &p)
	}

	return participants, nil
}

// UpdateParticipantStatus updates participant status
func (r *conferenceRepository) UpdateParticipantStatus(ctx context.Context, id uuid.UUID, status model.ParticipantStatus, leftAt *time.Time) error {
	query := `
		UPDATE con_test.conference_participants
		SET status = $2, left_at = $3
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, id, status, leftAt)
	if err != nil {
		return fmt.Errorf("failed to update participant status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// UpdateParticipantMute updates participant mute status
func (r *conferenceRepository) UpdateParticipantMute(ctx context.Context, id uuid.UUID, muted bool) error {
	query := `
		UPDATE con_test.conference_participants
		SET is_muted = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, muted)
	if err != nil {
		return fmt.Errorf("failed to update participant mute: %w", err)
	}

	return nil
}

// UpdateParticipantSpeaking updates participant speaking status
func (r *conferenceRepository) UpdateParticipantSpeaking(ctx context.Context, id uuid.UUID, speaking bool) error {
	query := `
		UPDATE con_test.conference_participants
		SET is_speaking = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, speaking)
	if err != nil {
		return fmt.Errorf("failed to update participant speaking: %w", err)
	}

	return nil
}

// SetParticipantFSMemberID sets the FreeSWITCH member ID for a participant
func (r *conferenceRepository) SetParticipantFSMemberID(ctx context.Context, id uuid.UUID, fsMemberID string) error {
	query := `
		UPDATE con_test.conference_participants
		SET fs_member_id = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, fsMemberID)
	if err != nil {
		return fmt.Errorf("failed to set participant FS member ID: %w", err)
	}

	return nil
}

// SetParticipantChannelUUID sets the FreeSWITCH channel UUID for a participant
func (r *conferenceRepository) SetParticipantChannelUUID(ctx context.Context, id uuid.UUID, channelUUID string) error {
	query := `
		UPDATE con_test.conference_participants
		SET channel_uuid = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, channelUUID)
	if err != nil {
		return fmt.Errorf("failed to set participant channel UUID: %w", err)
	}

	return nil
}

// GetParticipantByChannelUUID finds a participant by FreeSWITCH channel UUID
func (r *conferenceRepository) GetParticipantByChannelUUID(ctx context.Context, channelUUID string) (*model.Participant, error) {
	query := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.channel_uuid = $1 AND cp.left_at IS NULL
		LIMIT 1
	`

	var p model.Participant
	err := r.pool.QueryRow(ctx, query, channelUUID).Scan(
		&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
		&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
		&p.Role, &p.RSVPStatus, &p.RSVPAt,
		&p.Username, &p.DisplayName, &p.AvatarURL,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrParticipantNotFound
		}
		return nil, fmt.Errorf("failed to get participant by channel UUID: %w", err)
	}

	return &p, nil
}

// GetActiveParticipantCount returns the count of active participants in a conference
func (r *conferenceRepository) GetActiveParticipantCount(ctx context.Context, confID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM con_test.conference_participants
		WHERE conference_id = $1 AND status = 'joined'
	`

	var count int
	err := r.pool.QueryRow(ctx, query, confID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get participant count: %w", err)
	}

	return count, nil
}

// ======== Scheduled Events Operations ========

// CreateScheduledConference creates a scheduled conference with optional recurrence
func (r *conferenceRepository) CreateScheduledConference(ctx context.Context, conf *model.Conference, recurrence *model.RecurrenceRule) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO con_test.conferences (id, name, chat_id, freeswitch_name, created_by, status, max_members, is_private,
		                                   event_type, scheduled_at, series_id, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	now := time.Now()
	if conf.ID == uuid.Nil {
		conf.ID = uuid.New()
	}
	conf.CreatedAt = now
	conf.UpdatedAt = now

	_, err = tx.Exec(ctx, query,
		conf.ID, conf.Name, conf.ChatID, conf.FreeSwitchName, conf.CreatedBy,
		conf.Status, conf.MaxMembers, conf.IsPrivate, conf.EventType,
		conf.ScheduledAt, conf.SeriesID, conf.StartedAt, conf.CreatedAt, conf.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create scheduled conference: %w", err)
	}

	// Create recurrence rule if provided
	if recurrence != nil {
		recurrence.ConferenceID = conf.ID
		recurrence.ID = uuid.New()
		recurrence.CreatedAt = now

		recQuery := `
			INSERT INTO con_test.conference_recurrence (id, conference_id, frequency, days_of_week, day_of_month, until_date, occurrence_count, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.Exec(ctx, recQuery,
			recurrence.ID, recurrence.ConferenceID, recurrence.Frequency,
			recurrence.DaysOfWeek, recurrence.DayOfMonth, recurrence.UntilDate,
			recurrence.OccurrenceCount, recurrence.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to create recurrence rule: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListScheduledConferences lists scheduled conferences for a user
func (r *conferenceRepository) ListScheduledConferences(ctx context.Context, userID uuid.UUID, upcomingOnly bool, limit, offset int) ([]*model.Conference, int, error) {
	baseQuery := `
		SELECT DISTINCT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at, c.event_type, c.scheduled_at, c.series_id,
		       c.accepted_count, c.declined_count,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp2
		        WHERE cp2.conference_id = c.id) AS participant_count,
		       cp.role, cp.rsvp_status
		FROM con_test.conferences c
		INNER JOIN con_test.conference_participants cp ON c.id = cp.conference_id AND cp.user_id = $1
		WHERE c.event_type IN ('scheduled', 'recurring')
		  AND c.status IN ('scheduled', 'active')
	`

	countQuery := `
		SELECT COUNT(DISTINCT c.id)
		FROM con_test.conferences c
		INNER JOIN con_test.conference_participants cp ON c.id = cp.conference_id AND cp.user_id = $1
		WHERE c.event_type IN ('scheduled', 'recurring')
		  AND c.status IN ('scheduled', 'active')
	`

	if upcomingOnly {
		baseQuery += " AND c.scheduled_at >= NOW()"
		countQuery += " AND c.scheduled_at >= NOW()"
	}

	baseQuery += " ORDER BY c.scheduled_at ASC LIMIT $2 OFFSET $3"

	// Get total count
	var total int
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count scheduled conferences: %w", err)
	}

	// Get conferences
	rows, err := r.pool.Query(ctx, baseQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list scheduled conferences: %w", err)
	}
	defer rows.Close()

	conferences := make([]*model.Conference, 0)
	for rows.Next() {
		var conf model.Conference
		var userRole model.ConferenceRole
		var userRSVP model.RSVPStatus
		err := rows.Scan(
			&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
			&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
			&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
			&conf.EventType, &conf.ScheduledAt, &conf.SeriesID,
			&conf.AcceptedCount, &conf.DeclinedCount, &conf.ParticipantCount,
			&userRole, &userRSVP)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan scheduled conference: %w", err)
		}
		conferences = append(conferences, &conf)
	}

	// Fetch participants for each conference
	participantsQuery := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.conference_id = $1
		ORDER BY cp.role, cp.created_at
	`

	for _, conf := range conferences {
		pRows, err := r.pool.Query(ctx, participantsQuery, conf.ID)
		if err != nil {
			// Log but continue - don't fail the whole list
			continue
		}

		participants := make([]model.Participant, 0)
		for pRows.Next() {
			var p model.Participant
			err := pRows.Scan(
				&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
				&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
				&p.Role, &p.RSVPStatus, &p.RSVPAt,
				&p.Username, &p.DisplayName, &p.AvatarURL)
			if err != nil {
				continue
			}
			participants = append(participants, p)
		}
		pRows.Close()
		conf.Participants = participants
	}

	return conferences, total, nil
}

// GetChatConferences gets all conferences for a chat (for widget display)
func (r *conferenceRepository) GetChatConferences(ctx context.Context, chatID uuid.UUID, upcomingOnly bool) ([]*model.Conference, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at, c.event_type, c.scheduled_at, c.series_id,
		       c.accepted_count, c.declined_count,
		       (SELECT COUNT(*) FROM con_test.conference_participants cp
		        WHERE cp.conference_id = c.id) AS participant_count
		FROM con_test.conferences c
		WHERE c.chat_id = $1
		  AND c.status IN ('scheduled', 'active')
	`

	if upcomingOnly {
		query += " AND (c.scheduled_at >= NOW() OR c.event_type IN ('adhoc', 'adhoc_chat'))"
	}

	query += ` ORDER BY
		CASE WHEN c.event_type IN ('adhoc', 'adhoc_chat') AND c.status = 'active' THEN 0 ELSE 1 END,
		c.scheduled_at ASC`

	rows, err := r.pool.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat conferences: %w", err)
	}
	defer rows.Close()

	conferences := make([]*model.Conference, 0)
	for rows.Next() {
		var conf model.Conference
		err := rows.Scan(
			&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
			&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
			&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
			&conf.EventType, &conf.ScheduledAt, &conf.SeriesID,
			&conf.AcceptedCount, &conf.DeclinedCount, &conf.ParticipantCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat conference: %w", err)
		}
		conferences = append(conferences, &conf)
	}

	return conferences, nil
}

// UpdateParticipantRSVP updates participant's RSVP status
func (r *conferenceRepository) UpdateParticipantRSVP(ctx context.Context, participantID uuid.UUID, status model.RSVPStatus) error {
	query := `
		UPDATE con_test.conference_participants
		SET rsvp_status = $2, rsvp_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, participantID, status)
	if err != nil {
		return fmt.Errorf("failed to update participant RSVP: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// UpdateParticipantRole updates participant's role
func (r *conferenceRepository) UpdateParticipantRole(ctx context.Context, participantID uuid.UUID, role model.ConferenceRole) error {
	query := `
		UPDATE con_test.conference_participants
		SET role = $2
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, participantID, role)
	if err != nil {
		return fmt.Errorf("failed to update participant role: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// AddParticipantWithRole adds a participant with a specific role
func (r *conferenceRepository) AddParticipantWithRole(ctx context.Context, p *model.Participant) error {
	query := `
		INSERT INTO con_test.conference_participants (id, conference_id, user_id, status, is_muted, is_deaf, role, rsvp_status, joined_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = now
	if p.Role == "" {
		p.Role = model.RoleParticipant
	}
	if p.RSVPStatus == "" {
		p.RSVPStatus = model.RSVPPending
	}

	_, err := r.pool.Exec(ctx, query,
		p.ID, p.ConferenceID, p.UserID, p.Status, p.IsMuted, p.IsDeaf,
		p.Role, p.RSVPStatus, p.JoinedAt, p.CreatedAt)
	if err != nil {
		existing, _ := r.GetParticipant(ctx, p.ConferenceID, p.UserID)
		if existing != nil {
			return ErrAlreadyParticipant
		}
		return fmt.Errorf("failed to add participant with role: %w", err)
	}

	return nil
}

// RemoveParticipant removes a participant from a conference
func (r *conferenceRepository) RemoveParticipant(ctx context.Context, confID, userID uuid.UUID) error {
	query := `
		DELETE FROM con_test.conference_participants
		WHERE conference_id = $1 AND user_id = $2
	`

	result, err := r.pool.Exec(ctx, query, confID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrParticipantNotFound
	}

	return nil
}

// GetConferenceWithParticipants gets a conference with all its participants
func (r *conferenceRepository) GetConferenceWithParticipants(ctx context.Context, id uuid.UUID) (*model.Conference, error) {
	// Get conference
	conf, err := r.GetConference(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get all participants (including those who haven't joined yet for scheduled events)
	query := `
		SELECT cp.id, cp.conference_id, cp.user_id, cp.fs_member_id, cp.channel_uuid, cp.status,
		       cp.is_muted, cp.is_deaf, cp.is_speaking, cp.joined_at, cp.left_at, cp.created_at,
		       cp.role, cp.rsvp_status, cp.rsvp_at,
		       u.username, u.display_name, u.avatar_url
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON cp.user_id = u.id
		WHERE cp.conference_id = $1
		ORDER BY cp.role, cp.created_at
	`

	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get conference participants: %w", err)
	}
	defer rows.Close()

	participants := make([]model.Participant, 0)
	for rows.Next() {
		var p model.Participant
		err := rows.Scan(
			&p.ID, &p.ConferenceID, &p.UserID, &p.FSMemberID, &p.ChannelUUID, &p.Status,
			&p.IsMuted, &p.IsDeaf, &p.IsSpeaking, &p.JoinedAt, &p.LeftAt, &p.CreatedAt,
			&p.Role, &p.RSVPStatus, &p.RSVPAt,
			&p.Username, &p.DisplayName, &p.AvatarURL)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	conf.Participants = participants
	return conf, nil
}

// ======== Recurrence Operations ========

// CreateRecurrenceRule creates a recurrence rule for a conference
func (r *conferenceRepository) CreateRecurrenceRule(ctx context.Context, rule *model.RecurrenceRule) error {
	query := `
		INSERT INTO con_test.conference_recurrence (id, conference_id, frequency, days_of_week, day_of_month, until_date, occurrence_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
	}
	rule.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		rule.ID, rule.ConferenceID, rule.Frequency, rule.DaysOfWeek,
		rule.DayOfMonth, rule.UntilDate, rule.OccurrenceCount, rule.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create recurrence rule: %w", err)
	}

	return nil
}

// GetRecurrenceRule gets the recurrence rule for a conference
func (r *conferenceRepository) GetRecurrenceRule(ctx context.Context, conferenceID uuid.UUID) (*model.RecurrenceRule, error) {
	query := `
		SELECT id, conference_id, frequency, days_of_week, day_of_month, until_date, occurrence_count, created_at
		FROM con_test.conference_recurrence
		WHERE conference_id = $1
	`

	var rule model.RecurrenceRule
	err := r.pool.QueryRow(ctx, query, conferenceID).Scan(
		&rule.ID, &rule.ConferenceID, &rule.Frequency, &rule.DaysOfWeek,
		&rule.DayOfMonth, &rule.UntilDate, &rule.OccurrenceCount, &rule.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No recurrence rule is valid
		}
		return nil, fmt.Errorf("failed to get recurrence rule: %w", err)
	}

	return &rule, nil
}

// ======== Reminder Operations ========

// CreateReminder creates a reminder for a conference participant
func (r *conferenceRepository) CreateReminder(ctx context.Context, reminder *model.ConferenceReminder) error {
	query := `
		INSERT INTO con_test.conference_reminders (id, conference_id, user_id, remind_at, minutes_before, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	if reminder.ID == uuid.Nil {
		reminder.ID = uuid.New()
	}
	reminder.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		reminder.ID, reminder.ConferenceID, reminder.UserID,
		reminder.RemindAt, reminder.MinutesBefore, reminder.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create reminder: %w", err)
	}

	return nil
}

// GetPendingReminders gets all pending reminders that should be sent
func (r *conferenceRepository) GetPendingReminders(ctx context.Context, now time.Time) ([]*model.ConferenceReminder, error) {
	query := `
		SELECT r.id, r.conference_id, r.user_id, r.remind_at, r.minutes_before,
		       r.sent, r.sent_at, r.created_at, c.name, c.scheduled_at
		FROM con_test.conference_reminders r
		INNER JOIN con_test.conferences c ON r.conference_id = c.id
		WHERE r.sent = FALSE
		  AND r.remind_at <= $1
		  AND c.status = 'scheduled'
	`

	rows, err := r.pool.Query(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending reminders: %w", err)
	}
	defer rows.Close()

	reminders := make([]*model.ConferenceReminder, 0)
	for rows.Next() {
		var reminder model.ConferenceReminder
		err := rows.Scan(
			&reminder.ID, &reminder.ConferenceID, &reminder.UserID,
			&reminder.RemindAt, &reminder.MinutesBefore, &reminder.Sent,
			&reminder.SentAt, &reminder.CreatedAt, &reminder.ConferenceName, &reminder.ScheduledAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reminder: %w", err)
		}
		reminders = append(reminders, &reminder)
	}

	return reminders, nil
}

// MarkReminderSent marks a reminder as sent
func (r *conferenceRepository) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE con_test.conference_reminders
		SET sent = TRUE, sent_at = NOW()
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark reminder sent: %w", err)
	}

	return nil
}

// CleanupStaleConferences ends conferences that have been active for too long
// and have no active participants. Returns the number of conferences cleaned up.
func (r *conferenceRepository) CleanupStaleConferences(ctx context.Context, maxAge time.Duration) (int, error) {
	// First mark all participants as 'left' for stale conferences
	markParticipantsQuery := `
		UPDATE con_test.conference_participants
		SET status = 'left', left_at = NOW()
		WHERE conference_id IN (
			SELECT c.id FROM con_test.conferences c
			WHERE c.status = 'active'
			  AND c.created_at < NOW() - $1::interval
		)
		AND status = 'joined'
	`

	// Then end the conferences that have no active participants
	endConferencesQuery := `
		UPDATE con_test.conferences
		SET status = 'ended', ended_at = NOW()
		WHERE status = 'active'
		  AND (
		    -- Old conferences (older than maxAge)
		    created_at < NOW() - $1::interval
		    OR
		    -- Conferences with no active participants (empty for more than 5 minutes)
		    (
		      started_at IS NOT NULL
		      AND started_at < NOW() - INTERVAL '5 minutes'
		      AND (SELECT COUNT(*) FROM con_test.conference_participants cp
		           WHERE cp.conference_id = conferences.id AND cp.status = 'joined') = 0
		    )
		  )
	`

	// Execute in a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Mark stale participants as left
	_, err = tx.Exec(ctx, markParticipantsQuery, maxAge.String())
	if err != nil {
		return 0, fmt.Errorf("failed to mark stale participants: %w", err)
	}

	// End stale conferences
	result, err := tx.Exec(ctx, endConferencesQuery, maxAge.String())
	if err != nil {
		return 0, fmt.Errorf("failed to end stale conferences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// CleanupStaleConnectingParticipants marks participants stuck in 'connecting' status as 'left'
// if they have been in that status longer than the specified timeout.
// Returns the number of participants cleaned up.
func (r *conferenceRepository) CleanupStaleConnectingParticipants(ctx context.Context, timeout time.Duration) (int, error) {
	query := `
		UPDATE con_test.conference_participants
		SET status = 'left', left_at = NOW()
		WHERE status = 'connecting'
		  AND created_at < NOW() - $1::interval
		  AND conference_id IN (
		      SELECT id FROM con_test.conferences WHERE status = 'active'
		  )
	`

	result, err := r.pool.Exec(ctx, query, timeout.String())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale connecting participants: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// ======== History Operations ========

// LogModeratorAction logs a moderator action in a conference
func (r *conferenceRepository) LogModeratorAction(ctx context.Context, action *model.ModeratorAction) error {
	query := `
		INSERT INTO con_test.conference_moderator_actions (id, conference_id, actor_id, target_user_id, action_type, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	if action.ID == uuid.Nil {
		action.ID = uuid.New()
	}
	action.CreatedAt = time.Now()

	if action.Details == nil {
		action.Details = []byte("{}")
	}

	_, err := r.pool.Exec(ctx, query,
		action.ID, action.ConferenceID, action.ActorID, action.TargetUserID,
		action.ActionType, action.Details, action.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to log moderator action: %w", err)
	}

	return nil
}

// ListModeratorActions lists all moderator actions for a conference
func (r *conferenceRepository) ListModeratorActions(ctx context.Context, conferenceID uuid.UUID) ([]model.ModeratorAction, error) {
	query := `
		SELECT cma.id, cma.conference_id, cma.actor_id, cma.target_user_id,
		       cma.action_type, cma.details, cma.created_at,
		       actor.username AS actor_username, actor.display_name AS actor_display_name,
		       target.username AS target_username, target.display_name AS target_display_name
		FROM con_test.conference_moderator_actions cma
		LEFT JOIN con_test.users actor ON actor.id = cma.actor_id
		LEFT JOIN con_test.users target ON target.id = cma.target_user_id
		WHERE cma.conference_id = $1
		ORDER BY cma.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, conferenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list moderator actions: %w", err)
	}
	defer rows.Close()

	actions := make([]model.ModeratorAction, 0)
	for rows.Next() {
		var action model.ModeratorAction
		err := rows.Scan(
			&action.ID, &action.ConferenceID, &action.ActorID, &action.TargetUserID,
			&action.ActionType, &action.Details, &action.CreatedAt,
			&action.ActorUsername, &action.ActorDisplayName,
			&action.TargetUsername, &action.TargetDisplayName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan moderator action: %w", err)
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// ListConferenceHistory lists conference history for a chat
func (r *conferenceRepository) ListConferenceHistory(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*model.ConferenceHistory, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM con_test.conferences
		WHERE chat_id = $1 AND status IN ('ended', 'active')
	`

	var total int
	err := r.pool.QueryRow(ctx, countQuery, chatID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count conference history: %w", err)
	}

	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at, c.event_type, c.scheduled_at, c.series_id,
		       c.accepted_count, c.declined_count, c.thread_id,
		       COUNT(DISTINCT cp.user_id) AS participant_count
		FROM con_test.conferences c
		LEFT JOIN con_test.conference_participants cp ON cp.conference_id = c.id
		WHERE c.chat_id = $1 AND c.status IN ('ended', 'active')
		GROUP BY c.id
		ORDER BY c.started_at DESC NULLS LAST, c.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conference history: %w", err)
	}
	defer rows.Close()

	conferences := make([]*model.ConferenceHistory, 0)
	for rows.Next() {
		var conf model.ConferenceHistory
		err := rows.Scan(
			&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
			&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
			&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
			&conf.EventType, &conf.ScheduledAt, &conf.SeriesID,
			&conf.AcceptedCount, &conf.DeclinedCount, &conf.ThreadID,
			&conf.ParticipantCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan conference history: %w", err)
		}
		conferences = append(conferences, &conf)
	}

	return conferences, total, nil
}

// GetConferenceHistory gets detailed history for a specific conference
func (r *conferenceRepository) GetConferenceHistory(ctx context.Context, conferenceID uuid.UUID) (*model.ConferenceHistory, error) {
	query := `
		SELECT c.id, c.name, c.chat_id, c.freeswitch_name, c.created_by, c.status,
		       c.max_members, c.is_private, c.recording_path, c.started_at, c.ended_at,
		       c.created_at, c.updated_at, c.event_type, c.scheduled_at, c.series_id,
		       c.accepted_count, c.declined_count, c.thread_id,
		       COUNT(DISTINCT cp.user_id) AS participant_count
		FROM con_test.conferences c
		LEFT JOIN con_test.conference_participants cp ON cp.conference_id = c.id
		WHERE c.id = $1
		GROUP BY c.id
	`

	var conf model.ConferenceHistory
	err := r.pool.QueryRow(ctx, query, conferenceID).Scan(
		&conf.ID, &conf.Name, &conf.ChatID, &conf.FreeSwitchName, &conf.CreatedBy,
		&conf.Status, &conf.MaxMembers, &conf.IsPrivate, &conf.RecordingPath,
		&conf.StartedAt, &conf.EndedAt, &conf.CreatedAt, &conf.UpdatedAt,
		&conf.EventType, &conf.ScheduledAt, &conf.SeriesID,
		&conf.AcceptedCount, &conf.DeclinedCount, &conf.ThreadID,
		&conf.ParticipantCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConferenceNotFound
		}
		return nil, fmt.Errorf("failed to get conference history: %w", err)
	}

	// Get all participant sessions
	participants, err := r.GetAllParticipantSessions(ctx, conferenceID)
	if err != nil {
		return nil, err
	}
	conf.AllParticipants = participants

	// Get moderator actions
	actions, err := r.ListModeratorActions(ctx, conferenceID)
	if err != nil {
		return nil, err
	}
	conf.ModeratorActions = actions

	return &conf, nil
}

// GetAllParticipantSessions gets all participant sessions grouped by user
func (r *conferenceRepository) GetAllParticipantSessions(ctx context.Context, conferenceID uuid.UUID) ([]model.ParticipantHistory, error) {
	query := `
		SELECT cp.user_id, u.username, u.display_name, u.avatar_url,
		       cp.joined_at, cp.left_at, cp.status, cp.role
		FROM con_test.conference_participants cp
		LEFT JOIN con_test.users u ON u.id = cp.user_id
		WHERE cp.conference_id = $1
		ORDER BY cp.user_id, cp.joined_at
	`

	rows, err := r.pool.Query(ctx, query, conferenceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participant sessions: %w", err)
	}
	defer rows.Close()

	// Group by user
	userSessions := make(map[uuid.UUID]*model.ParticipantHistory)
	var userOrder []uuid.UUID

	for rows.Next() {
		var userID uuid.UUID
		var username, displayName, avatarURL *string
		var joinedAt *time.Time
		var leftAt *time.Time
		var status model.ParticipantStatus
		var role model.ConferenceRole

		err := rows.Scan(&userID, &username, &displayName, &avatarURL,
			&joinedAt, &leftAt, &status, &role)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant session: %w", err)
		}

		history, exists := userSessions[userID]
		if !exists {
			history = &model.ParticipantHistory{
				UserID:      userID,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Sessions:    make([]model.ParticipantSession, 0),
			}
			userSessions[userID] = history
			userOrder = append(userOrder, userID)
		}

		if joinedAt != nil {
			session := model.ParticipantSession{
				JoinedAt: *joinedAt,
				LeftAt:   leftAt,
				Status:   status,
				Role:     role,
			}
			history.Sessions = append(history.Sessions, session)
		}
	}

	// Convert to slice maintaining order
	result := make([]model.ParticipantHistory, 0, len(userOrder))
	for _, userID := range userOrder {
		result = append(result, *userSessions[userID])
	}

	return result, nil
}

// SetConferenceThread sets the thread_id for a conference
func (r *conferenceRepository) SetConferenceThread(ctx context.Context, conferenceID, threadID uuid.UUID) error {
	query := `
		UPDATE con_test.conferences
		SET thread_id = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.pool.Exec(ctx, query, conferenceID, threadID)
	if err != nil {
		return fmt.Errorf("failed to set conference thread: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrConferenceNotFound
	}

	return nil
}

// GetConferenceThread gets the thread_id for a conference
func (r *conferenceRepository) GetConferenceThread(ctx context.Context, conferenceID uuid.UUID) (*uuid.UUID, error) {
	query := `
		SELECT thread_id FROM con_test.conferences WHERE id = $1
	`

	var threadID *uuid.UUID
	err := r.pool.QueryRow(ctx, query, conferenceID).Scan(&threadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrConferenceNotFound
		}
		return nil, fmt.Errorf("failed to get conference thread: %w", err)
	}

	return threadID, nil
}

// GetConferenceMessages retrieves messages from a chat during a conference time window
func (r *conferenceRepository) GetConferenceMessages(ctx context.Context, confID, chatID uuid.UUID, startedAt, endedAt *time.Time) ([]*model.ConferenceMessage, error) {
	// If conference hasn't started yet, no messages
	if startedAt == nil {
		return []*model.ConferenceMessage{}, nil
	}

	// Build query - get messages during conference time window
	// If endedAt is nil (active conference), get all messages from startedAt until now
	query := `
		SELECT
			m.id,
			m.chat_id,
			m.sender_id,
			m.content,
			m.created_at,
			u.username as sender_username,
			u.display_name as sender_display_name
		FROM con_test.messages m
		LEFT JOIN con_test.users u ON m.sender_id = u.id
		WHERE m.chat_id = $1
		  AND m.created_at >= $2
		  AND ($3::timestamptz IS NULL OR m.created_at <= $3)
		  AND m.is_deleted = false
		ORDER BY m.created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, chatID, startedAt, endedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to query conference messages: %w", err)
	}
	defer rows.Close()

	var messages []*model.ConferenceMessage
	for rows.Next() {
		msg := &model.ConferenceMessage{}
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Content,
			&msg.CreatedAt,
			&msg.SenderUsername,
			&msg.SenderDisplayName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}
