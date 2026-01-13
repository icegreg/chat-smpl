package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/icegreg/chat-smpl/services/admin/internal/client"
	"github.com/icegreg/chat-smpl/services/admin/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ConferenceService handles conference operations
type ConferenceService struct {
	db          *pgxpool.Pool
	voiceClient *client.VoiceClient
	logger      *zap.Logger
}

// NewConferenceService creates a new conference service
func NewConferenceService(db *pgxpool.Pool, voiceClient *client.VoiceClient, logger *zap.Logger) *ConferenceService {
	return &ConferenceService{
		db:          db,
		voiceClient: voiceClient,
		logger:      logger,
	}
}

// ListConferences returns all conferences with optional filters
func (s *ConferenceService) ListConferences(ctx context.Context, status *model.ConferenceStatus) ([]model.Conference, error) {
	query := `
		SELECT
			c.id,
			c.name,
			c.event_type,
			c.chat_id,
			c.status,
			c.created_by,
			c.created_at,
			c.started_at,
			c.ended_at,
			EXTRACT(EPOCH FROM (COALESCE(c.ended_at, NOW()) - c.started_at))::bigint as duration,
			COUNT(DISTINCT p.id) FILTER (WHERE p.status = 'connected') as participant_count
		FROM voice.conferences c
		LEFT JOIN voice.participants p ON c.id = p.conference_id
	`

	args := []interface{}{}
	if status != nil {
		query += " WHERE c.status = $1"
		args = append(args, string(*status))
	}

	query += " GROUP BY c.id ORDER BY c.created_at DESC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		s.logger.Error("failed to query conferences", zap.Error(err))
		return nil, fmt.Errorf("failed to query conferences: %w", err)
	}
	defer rows.Close()

	var conferences []model.Conference
	for rows.Next() {
		var conf model.Conference
		err := rows.Scan(
			&conf.ID,
			&conf.Name,
			&conf.EventType,
			&conf.ChatID,
			&conf.Status,
			&conf.CreatedBy,
			&conf.CreatedAt,
			&conf.StartedAt,
			&conf.EndedAt,
			&conf.Duration,
			&conf.Participants,
		)
		if err != nil {
			s.logger.Error("failed to scan conference row", zap.Error(err))
			continue
		}
		conferences = append(conferences, conf)
	}

	return conferences, nil
}

// GetConference returns a single conference by ID
func (s *ConferenceService) GetConference(ctx context.Context, id uuid.UUID) (*model.Conference, error) {
	query := `
		SELECT
			c.id,
			c.name,
			c.event_type,
			c.chat_id,
			c.status,
			c.created_by,
			c.created_at,
			c.started_at,
			c.ended_at,
			EXTRACT(EPOCH FROM (COALESCE(c.ended_at, NOW()) - c.started_at))::bigint as duration,
			COUNT(DISTINCT p.id) FILTER (WHERE p.status = 'connected') as participant_count
		FROM voice.conferences c
		LEFT JOIN voice.participants p ON c.id = p.conference_id
		WHERE c.id = $1
		GROUP BY c.id
	`

	var conf model.Conference
	err := s.db.QueryRow(ctx, query, id).Scan(
		&conf.ID,
		&conf.Name,
		&conf.EventType,
		&conf.ChatID,
		&conf.Status,
		&conf.CreatedBy,
		&conf.CreatedAt,
		&conf.StartedAt,
		&conf.EndedAt,
		&conf.Duration,
		&conf.Participants,
	)
	if err != nil {
		s.logger.Error("failed to get conference", zap.Error(err), zap.String("id", id.String()))
		return nil, fmt.Errorf("failed to get conference: %w", err)
	}

	return &conf, nil
}

// ListParticipants returns participants of a conference
func (s *ConferenceService) ListParticipants(ctx context.Context, conferenceID uuid.UUID) ([]model.Participant, error) {
	query := `
		SELECT
			p.id,
			p.conference_id,
			p.user_id,
			u.username,
			u.extension,
			p.status,
			p.joined_at,
			p.left_at,
			EXTRACT(EPOCH FROM (COALESCE(p.left_at, NOW()) - p.joined_at))::bigint as duration
		FROM voice.participants p
		JOIN con_test.users u ON p.user_id = u.user_guid
		WHERE p.conference_id = $1
		ORDER BY p.joined_at DESC
	`

	rows, err := s.db.Query(ctx, query, conferenceID)
	if err != nil {
		s.logger.Error("failed to query participants", zap.Error(err), zap.String("conference_id", conferenceID.String()))
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	var participants []model.Participant
	for rows.Next() {
		var p model.Participant
		err := rows.Scan(
			&p.ID,
			&p.ConferenceID,
			&p.UserID,
			&p.Username,
			&p.Extension,
			&p.Status,
			&p.JoinedAt,
			&p.LeftAt,
			&p.Duration,
		)
		if err != nil {
			s.logger.Error("failed to scan participant row", zap.Error(err))
			continue
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// EndConference ends a conference
func (s *ConferenceService) EndConference(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Use voice service to end the conference
	err := s.voiceClient.EndConference(ctx, id.String(), userID.String())
	if err != nil {
		return fmt.Errorf("failed to end conference via voice service: %w", err)
	}

	return nil
}

// GetActiveConferencesCount returns count of active conferences
func (s *ConferenceService) GetActiveConferencesCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM voice.conferences WHERE status = 'active'
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active conferences count: %w", err)
	}
	return count, nil
}
