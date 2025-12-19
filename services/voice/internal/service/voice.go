package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/icegreg/chat-smpl/services/voice/internal/config"
	"github.com/icegreg/chat-smpl/services/voice/internal/esl"
	"github.com/icegreg/chat-smpl/services/voice/internal/events"
	"github.com/icegreg/chat-smpl/services/voice/internal/model"
	"github.com/icegreg/chat-smpl/services/voice/internal/repository"
)

// VoiceService interface defines voice/conference operations
type VoiceService interface {
	// Conferences
	CreateConference(ctx context.Context, req *model.CreateConferenceRequest) (*model.Conference, error)
	GetConference(ctx context.Context, confID uuid.UUID) (*model.Conference, error)
	ListConferences(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*model.Conference, int, error)
	JoinConference(ctx context.Context, confID, userID uuid.UUID, opts model.JoinOptions) (*model.Participant, error)
	LeaveConference(ctx context.Context, confID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, confID uuid.UUID) ([]*model.Participant, error)
	MuteParticipant(ctx context.Context, confID, targetUserID, actorUserID uuid.UUID, mute bool) (*model.Participant, error)
	KickParticipant(ctx context.Context, confID, targetUserID, actorUserID uuid.UUID) error
	EndConference(ctx context.Context, confID, userID uuid.UUID) error

	// Direct calls (1-on-1)
	InitiateCall(ctx context.Context, callerID, calleeID uuid.UUID, chatID *uuid.UUID) (*model.Call, error)
	AnswerCall(ctx context.Context, callID, userID uuid.UUID) (*model.Call, error)
	HangupCall(ctx context.Context, callID, userID uuid.UUID) error
	GetCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Call, int, error)

	// Auth & Quick actions
	GetVertoCredentials(ctx context.Context, userID uuid.UUID) (*model.VertoCredentials, error)
	StartChatCall(ctx context.Context, chatID, userID uuid.UUID) (*model.Conference, *model.VertoCredentials, error)

	// Scheduled events
	ScheduleConference(ctx context.Context, req *model.ScheduleConferenceRequest) (*model.Conference, error)
	CreateAdHocFromChat(ctx context.Context, req *model.CreateAdHocFromChatRequest) (*model.Conference, error)
	CreateQuickAdHoc(ctx context.Context, userID uuid.UUID, name string) (*model.Conference, error)
	UpdateRSVP(ctx context.Context, req *model.UpdateRSVPRequest) (*model.Participant, error)
	UpdateParticipantRole(ctx context.Context, req *model.UpdateParticipantRoleRequest) (*model.Participant, error)
	AddParticipants(ctx context.Context, req *model.AddParticipantsRequest) error
	RemoveParticipant(ctx context.Context, req *model.RemoveParticipantRequest) error
	ListScheduledConferences(ctx context.Context, userID uuid.UUID, upcomingOnly bool, limit, offset int) ([]*model.Conference, int, error)
	GetChatConferences(ctx context.Context, chatID uuid.UUID, upcomingOnly bool) ([]*model.Conference, error)
	CancelConference(ctx context.Context, confID, userID uuid.UUID, cancelSeries bool) error
}

type voiceService struct {
	cfg            *config.Config
	eslClient      esl.Client
	confRepo       repository.ConferenceRepository
	callRepo       repository.CallRepository
	eventPublisher events.Publisher
	logger         *zap.Logger
}

// NewVoiceService creates a new voice service instance
func NewVoiceService(
	cfg *config.Config,
	eslClient esl.Client,
	confRepo repository.ConferenceRepository,
	callRepo repository.CallRepository,
	eventPublisher events.Publisher,
	logger *zap.Logger,
) VoiceService {
	return &voiceService{
		cfg:            cfg,
		eslClient:      eslClient,
		confRepo:       confRepo,
		callRepo:       callRepo,
		eventPublisher: eventPublisher,
		logger:         logger,
	}
}

// CreateConference creates a new conference room
func (s *voiceService) CreateConference(ctx context.Context, req *model.CreateConferenceRequest) (*model.Conference, error) {
	// Generate FreeSWITCH conference name
	fsName := fmt.Sprintf("conf_%s", uuid.New().String()[:8])
	if req.IsPrivate {
		fsName = fmt.Sprintf("private_%s", uuid.New().String())
	}

	conf := &model.Conference{
		Name:           req.Name,
		ChatID:         req.ChatID,
		FreeSwitchName: fsName,
		CreatedBy:      req.CreatedBy,
		Status:         model.ConferenceStatusActive,
		MaxMembers:     req.MaxMembers,
		IsPrivate:      req.IsPrivate,
	}

	if conf.MaxMembers == 0 {
		conf.MaxMembers = 10
	}

	// Create conference in FreeSWITCH
	profile := "default"
	if !req.EnableRecording {
		profile = "norecord"
	}
	if req.IsPrivate {
		profile = "private"
	}

	if err := s.eslClient.CreateConference(ctx, fsName, profile); err != nil {
		s.logger.Error("failed to create conference in FreeSWITCH", zap.Error(err))
		// Continue anyway - FreeSWITCH creates conferences dynamically
	}

	// Save to database
	if err := s.confRepo.CreateConference(ctx, conf); err != nil {
		return nil, fmt.Errorf("failed to create conference: %w", err)
	}

	// Publish event
	if err := s.eventPublisher.PublishConferenceCreated(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.created event", zap.Error(err))
	}

	s.logger.Info("conference created",
		zap.String("id", conf.ID.String()),
		zap.String("name", conf.Name),
		zap.String("fsName", conf.FreeSwitchName))

	return conf, nil
}

// GetConference retrieves a conference by ID
func (s *voiceService) GetConference(ctx context.Context, confID uuid.UUID) (*model.Conference, error) {
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return nil, err
	}

	// Get participant count from FreeSWITCH if active
	if conf.Status == model.ConferenceStatusActive && s.eslClient.IsConnected() {
		members, err := s.eslClient.GetConferenceMembers(ctx, conf.FreeSwitchName)
		if err == nil {
			conf.ParticipantCount = len(members)
		}
	}

	return conf, nil
}

// ListConferences lists conferences for a user
func (s *voiceService) ListConferences(ctx context.Context, userID uuid.UUID, activeOnly bool, limit, offset int) ([]*model.Conference, int, error) {
	return s.confRepo.ListConferences(ctx, userID, activeOnly, limit, offset)
}

// JoinConference adds a user to a conference
func (s *voiceService) JoinConference(ctx context.Context, confID, userID uuid.UUID, opts model.JoinOptions) (*model.Participant, error) {
	// Get conference
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return nil, err
	}

	if conf.Status != model.ConferenceStatusActive {
		return nil, fmt.Errorf("conference is not active")
	}

	// Check max members
	participants, err := s.confRepo.ListParticipants(ctx, confID)
	if err != nil {
		return nil, err
	}

	if len(participants) >= conf.MaxMembers {
		return nil, fmt.Errorf("conference is full")
	}

	// Check if user is already in conference
	for _, p := range participants {
		if p.UserID == userID {
			return nil, fmt.Errorf("user already in conference")
		}
	}

	// Create participant record
	participant := &model.Participant{
		ConferenceID: confID,
		UserID:       userID,
		Status:       model.ParticipantStatusConnecting,
		IsMuted:      opts.Muted,
	}

	if err := s.confRepo.AddParticipant(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	// Publish event
	if err := s.eventPublisher.PublishParticipantJoined(ctx, participant); err != nil {
		s.logger.Error("failed to publish participant.joined event", zap.Error(err))
	}

	s.logger.Info("participant joined conference",
		zap.String("conferenceId", confID.String()),
		zap.String("userId", userID.String()))

	return participant, nil
}

// LeaveConference removes a user from a conference
func (s *voiceService) LeaveConference(ctx context.Context, confID, userID uuid.UUID) error {
	// Get participant
	participant, err := s.confRepo.GetParticipant(ctx, confID, userID)
	if err != nil {
		return err
	}

	// Kick from FreeSWITCH if connected
	if participant.FSMemberID != nil && s.eslClient.IsConnected() {
		conf, _ := s.confRepo.GetConference(ctx, confID)
		if conf != nil {
			if err := s.eslClient.KickMember(ctx, conf.FreeSwitchName, *participant.FSMemberID); err != nil {
				s.logger.Warn("failed to kick member from FreeSWITCH", zap.Error(err))
			}
		}
	}

	// Update participant status
	leftAt := time.Now()
	if err := s.confRepo.UpdateParticipantStatus(ctx, participant.ID, model.ParticipantStatusLeft, &leftAt); err != nil {
		return err
	}

	participant.Status = model.ParticipantStatusLeft

	// Publish event
	if err := s.eventPublisher.PublishParticipantLeft(ctx, participant); err != nil {
		s.logger.Error("failed to publish participant.left event", zap.Error(err))
	}

	// Check if conference should end (no more active participants)
	activeParticipants, _ := s.confRepo.ListParticipants(ctx, confID)
	if len(activeParticipants) == 0 {
		s.logger.Info("conference has no participants, ending", zap.String("conferenceId", confID.String()))
		_ = s.endConferenceInternal(ctx, confID)
	}

	return nil
}

// GetParticipants returns all participants in a conference
func (s *voiceService) GetParticipants(ctx context.Context, confID uuid.UUID) ([]*model.Participant, error) {
	return s.confRepo.ListParticipants(ctx, confID)
}

// MuteParticipant mutes or unmutes a participant
func (s *voiceService) MuteParticipant(ctx context.Context, confID, targetUserID, actorUserID uuid.UUID, mute bool) (*model.Participant, error) {
	// Get conference
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return nil, err
	}

	// Check permissions: only creator or self can mute
	if actorUserID != targetUserID && actorUserID != conf.CreatedBy {
		return nil, fmt.Errorf("not authorized to mute this participant")
	}

	// Get participant
	participant, err := s.confRepo.GetParticipant(ctx, confID, targetUserID)
	if err != nil {
		return nil, err
	}

	// Mute in FreeSWITCH
	if participant.FSMemberID != nil && s.eslClient.IsConnected() {
		if err := s.eslClient.MuteMember(ctx, conf.FreeSwitchName, *participant.FSMemberID, mute); err != nil {
			s.logger.Warn("failed to mute member in FreeSWITCH", zap.Error(err))
		}
	}

	// Update database
	if err := s.confRepo.UpdateParticipantMute(ctx, participant.ID, mute); err != nil {
		return nil, err
	}

	participant.IsMuted = mute

	// Publish event
	if err := s.eventPublisher.PublishParticipantMuted(ctx, participant); err != nil {
		s.logger.Error("failed to publish participant.muted event", zap.Error(err))
	}

	return participant, nil
}

// KickParticipant removes a participant from conference (by moderator)
func (s *voiceService) KickParticipant(ctx context.Context, confID, targetUserID, actorUserID uuid.UUID) error {
	// Get conference
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return err
	}

	// Only creator can kick
	if actorUserID != conf.CreatedBy {
		return fmt.Errorf("not authorized to kick participants")
	}

	// Can't kick yourself
	if actorUserID == targetUserID {
		return fmt.Errorf("cannot kick yourself")
	}

	// Get participant
	participant, err := s.confRepo.GetParticipant(ctx, confID, targetUserID)
	if err != nil {
		return err
	}

	// Kick from FreeSWITCH
	if participant.FSMemberID != nil && s.eslClient.IsConnected() {
		if err := s.eslClient.KickMember(ctx, conf.FreeSwitchName, *participant.FSMemberID); err != nil {
			s.logger.Warn("failed to kick member from FreeSWITCH", zap.Error(err))
		}
	}

	// Update status
	kickedAt := time.Now()
	if err := s.confRepo.UpdateParticipantStatus(ctx, participant.ID, model.ParticipantStatusKicked, &kickedAt); err != nil {
		return err
	}

	participant.Status = model.ParticipantStatusKicked

	// Publish event
	if err := s.eventPublisher.PublishParticipantLeft(ctx, participant); err != nil {
		s.logger.Error("failed to publish participant.left event", zap.Error(err))
	}

	return nil
}

// EndConference ends a conference
func (s *voiceService) EndConference(ctx context.Context, confID, userID uuid.UUID) error {
	// Get conference
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return err
	}

	// Only creator can end
	if userID != conf.CreatedBy {
		return fmt.Errorf("not authorized to end conference")
	}

	return s.endConferenceInternal(ctx, confID)
}

func (s *voiceService) endConferenceInternal(ctx context.Context, confID uuid.UUID) error {
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return err
	}

	// Kick all members from FreeSWITCH
	if s.eslClient.IsConnected() {
		members, _ := s.eslClient.GetConferenceMembers(ctx, conf.FreeSwitchName)
		for _, m := range members {
			_ = s.eslClient.KickMember(ctx, conf.FreeSwitchName, m.ID)
		}
	}

	// Update all participants to left
	participants, _ := s.confRepo.ListParticipants(ctx, confID)
	leftAt := time.Now()
	for _, p := range participants {
		_ = s.confRepo.UpdateParticipantStatus(ctx, p.ID, model.ParticipantStatusLeft, &leftAt)
	}

	// Update conference status
	endedAt := time.Now()
	if err := s.confRepo.UpdateConferenceStatus(ctx, confID, model.ConferenceStatusEnded, &endedAt); err != nil {
		return err
	}

	conf.Status = model.ConferenceStatusEnded

	// Publish event
	if err := s.eventPublisher.PublishConferenceEnded(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.ended event", zap.Error(err))
	}

	s.logger.Info("conference ended", zap.String("conferenceId", confID.String()))

	return nil
}

// InitiateCall starts a 1-on-1 call
func (s *voiceService) InitiateCall(ctx context.Context, callerID, calleeID uuid.UUID, chatID *uuid.UUID) (*model.Call, error) {
	// Check if either user has an active call
	activeCall, _ := s.callRepo.GetActiveCallForUser(ctx, callerID)
	if activeCall != nil {
		return nil, fmt.Errorf("caller already has an active call")
	}

	activeCall, _ = s.callRepo.GetActiveCallForUser(ctx, calleeID)
	if activeCall != nil {
		return nil, fmt.Errorf("callee is already on a call")
	}

	call := &model.Call{
		CallerID: callerID,
		CalleeID: calleeID,
		ChatID:   chatID,
		Status:   model.CallStatusInitiated,
	}

	if err := s.callRepo.CreateCall(ctx, call); err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	// Publish event to notify callee
	if err := s.eventPublisher.PublishCallInitiated(ctx, call); err != nil {
		s.logger.Error("failed to publish call.initiated event", zap.Error(err))
	}

	s.logger.Info("call initiated",
		zap.String("callId", call.ID.String()),
		zap.String("callerId", callerID.String()),
		zap.String("calleeId", calleeID.String()))

	return call, nil
}

// AnswerCall answers an incoming call
func (s *voiceService) AnswerCall(ctx context.Context, callID, userID uuid.UUID) (*model.Call, error) {
	call, err := s.callRepo.GetCall(ctx, callID)
	if err != nil {
		return nil, err
	}

	// Only callee can answer
	if call.CalleeID != userID {
		return nil, fmt.Errorf("not authorized to answer this call")
	}

	// Check status
	if call.Status != model.CallStatusInitiated && call.Status != model.CallStatusRinging {
		return nil, fmt.Errorf("call cannot be answered in current state: %s", call.Status)
	}

	// Update call status
	answeredAt := time.Now()
	if err := s.callRepo.UpdateCallAnswered(ctx, callID, answeredAt); err != nil {
		return nil, err
	}

	call.Status = model.CallStatusAnswered
	call.AnsweredAt = &answeredAt

	// Publish event
	if err := s.eventPublisher.PublishCallAnswered(ctx, call); err != nil {
		s.logger.Error("failed to publish call.answered event", zap.Error(err))
	}

	s.logger.Info("call answered",
		zap.String("callId", callID.String()),
		zap.String("userId", userID.String()))

	return call, nil
}

// HangupCall ends a call
func (s *voiceService) HangupCall(ctx context.Context, callID, userID uuid.UUID) error {
	call, err := s.callRepo.GetCall(ctx, callID)
	if err != nil {
		return err
	}

	// Either party can hangup
	if call.CallerID != userID && call.CalleeID != userID {
		return fmt.Errorf("not authorized to hangup this call")
	}

	// Hangup in FreeSWITCH if active
	if call.FSCallUUID != nil && s.eslClient.IsConnected() {
		if err := s.eslClient.Hangup(ctx, *call.FSCallUUID, "NORMAL_CLEARING"); err != nil {
			s.logger.Warn("failed to hangup call in FreeSWITCH", zap.Error(err))
		}
	}

	// Calculate duration
	var duration int
	endedAt := time.Now()
	if call.AnsweredAt != nil {
		duration = int(endedAt.Sub(*call.AnsweredAt).Seconds())
	}

	endReason := "user_hangup"
	if call.Status == model.CallStatusInitiated || call.Status == model.CallStatusRinging {
		if userID == call.CalleeID {
			endReason = "rejected"
		} else {
			endReason = "cancelled"
		}
	}

	// Update call
	if err := s.callRepo.UpdateCallEnded(ctx, callID, endedAt, duration, endReason); err != nil {
		return err
	}

	call.Status = model.CallStatusEnded
	call.EndedAt = &endedAt
	call.Duration = duration
	call.EndReason = &endReason

	// Publish event
	if err := s.eventPublisher.PublishCallEnded(ctx, call); err != nil {
		s.logger.Error("failed to publish call.ended event", zap.Error(err))
	}

	s.logger.Info("call ended",
		zap.String("callId", callID.String()),
		zap.String("reason", endReason),
		zap.Int("duration", duration))

	return nil
}

// GetCallHistory returns call history for a user
func (s *voiceService) GetCallHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Call, int, error) {
	return s.callRepo.GetCallHistory(ctx, userID, limit, offset)
}

// GetVertoCredentials generates temporary Verto credentials for a user
func (s *voiceService) GetVertoCredentials(ctx context.Context, userID uuid.UUID) (*model.VertoCredentials, error) {
	// Generate random password
	passwordBytes := make([]byte, 16)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}
	password := hex.EncodeToString(passwordBytes)

	// Login format: user_<uuid>@domain
	login := fmt.Sprintf("user_%s@%s", userID.String()[:8], s.cfg.Verto.Domain)

	expiresAt := time.Now().Add(time.Duration(s.cfg.Verto.CredentialsTTL) * time.Second)

	creds := &model.VertoCredentials{
		UserID:    userID,
		Login:     login,
		Password:  password,
		WSUrl:     s.cfg.Verto.WSUrl,
		ExpiresAt: expiresAt.Unix(),
		IceServers: []model.IceServer{
			{
				URLs: s.cfg.TURN.URLs,
			},
		},
	}

	// Add TURN credentials if configured
	if s.cfg.TURN.Username != "" {
		creds.IceServers[0].Username = s.cfg.TURN.Username
		creds.IceServers[0].Credential = s.cfg.TURN.Credential
	}

	s.logger.Debug("generated verto credentials",
		zap.String("userId", userID.String()),
		zap.String("login", login))

	return creds, nil
}

// StartChatCall starts a quick call from a chat room
func (s *voiceService) StartChatCall(ctx context.Context, chatID, userID uuid.UUID) (*model.Conference, *model.VertoCredentials, error) {
	// Check if there's already an active conference for this chat
	existingConf, err := s.confRepo.GetConferenceByChatID(ctx, chatID)
	if err == nil && existingConf != nil && existingConf.Status == model.ConferenceStatusActive {
		// Return existing conference
		creds, err := s.GetVertoCredentials(ctx, userID)
		if err != nil {
			return nil, nil, err
		}
		return existingConf, creds, nil
	}

	// Create new conference for this chat
	conf, err := s.CreateConference(ctx, &model.CreateConferenceRequest{
		Name:            fmt.Sprintf("Chat call - %s", chatID.String()[:8]),
		ChatID:          &chatID,
		CreatedBy:       userID,
		MaxMembers:      10,
		IsPrivate:       false,
		EnableRecording: true,
	})
	if err != nil {
		return nil, nil, err
	}

	// Get credentials for user
	creds, err := s.GetVertoCredentials(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	return conf, creds, nil
}

// ======== Scheduled Events ========

// ScheduleConference creates a scheduled or recurring conference
func (s *voiceService) ScheduleConference(ctx context.Context, req *model.ScheduleConferenceRequest) (*model.Conference, error) {
	// Generate FreeSWITCH conference name
	fsName := fmt.Sprintf("scheduled_%s", uuid.New().String()[:8])

	eventType := model.EventTypeScheduled
	if req.Recurrence != nil {
		eventType = model.EventTypeRecurring
	}

	// For recurring events, generate series ID
	var seriesID *uuid.UUID
	if eventType == model.EventTypeRecurring {
		id := uuid.New()
		seriesID = &id
	}

	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 50
	}

	conf := &model.Conference{
		ID:             uuid.New(),
		Name:           req.Name,
		ChatID:         req.ChatID,
		FreeSwitchName: fsName,
		CreatedBy:      req.UserID,
		Status:         model.ConferenceStatusScheduled,
		MaxMembers:     maxMembers,
		IsPrivate:      false,
		EventType:      eventType,
		ScheduledAt:    &req.ScheduledAt,
		SeriesID:       seriesID,
	}

	// Create conference with optional recurrence
	if err := s.confRepo.CreateScheduledConference(ctx, conf, req.Recurrence); err != nil {
		return nil, fmt.Errorf("failed to create scheduled conference: %w", err)
	}

	// Add creator as originator
	creator := &model.Participant{
		ConferenceID: conf.ID,
		UserID:       req.UserID,
		Status:       model.ParticipantStatusConnecting,
		Role:         model.RoleOriginator,
		RSVPStatus:   model.RSVPAccepted, // Creator auto-accepts
	}
	if err := s.confRepo.AddParticipantWithRole(ctx, creator); err != nil {
		s.logger.Error("failed to add creator as participant", zap.Error(err))
	}

	// Add other participants
	for _, participantID := range req.ParticipantIDs {
		if participantID == req.UserID {
			continue // Skip creator
		}
		p := &model.Participant{
			ConferenceID: conf.ID,
			UserID:       participantID,
			Status:       model.ParticipantStatusConnecting,
			Role:         model.RoleParticipant,
			RSVPStatus:   model.RSVPPending,
		}
		if err := s.confRepo.AddParticipantWithRole(ctx, p); err != nil {
			s.logger.Warn("failed to add participant", zap.String("userId", participantID.String()), zap.Error(err))
		}

		// Create reminder for participant (15 minutes before)
		s.createReminder(ctx, conf.ID, participantID, req.ScheduledAt, 15)
	}

	// Create reminder for creator too
	s.createReminder(ctx, conf.ID, req.UserID, req.ScheduledAt, 15)

	// Get conference with participants for response
	conf, _ = s.confRepo.GetConferenceWithParticipants(ctx, conf.ID)

	// Publish event
	if err := s.eventPublisher.PublishConferenceScheduled(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.scheduled event", zap.Error(err))
	}

	s.logger.Info("scheduled conference created",
		zap.String("id", conf.ID.String()),
		zap.String("name", conf.Name),
		zap.String("eventType", string(eventType)),
		zap.Time("scheduledAt", req.ScheduledAt))

	return conf, nil
}

// CreateAdHocFromChat creates an ad-hoc call from a chat with selected participants
func (s *voiceService) CreateAdHocFromChat(ctx context.Context, req *model.CreateAdHocFromChatRequest) (*model.Conference, error) {
	now := time.Now()
	fsName := fmt.Sprintf("adhoc_chat_%s", uuid.New().String()[:8])

	conf := &model.Conference{
		ID:             uuid.New(),
		Name:           fmt.Sprintf("Ad-hoc call %s", now.Format("15:04")),
		ChatID:         &req.ChatID,
		FreeSwitchName: fsName,
		CreatedBy:      req.UserID,
		Status:         model.ConferenceStatusActive,
		MaxMembers:     50,
		IsPrivate:      false,
		EventType:      model.EventTypeAdhocChat,
		StartedAt:      &now,
	}

	// Create conference
	if err := s.confRepo.CreateScheduledConference(ctx, conf, nil); err != nil {
		return nil, fmt.Errorf("failed to create ad-hoc conference: %w", err)
	}

	// Create in FreeSWITCH
	if err := s.eslClient.CreateConference(ctx, fsName, "default"); err != nil {
		s.logger.Warn("failed to create conference in FreeSWITCH", zap.Error(err))
	}

	// Add creator as originator
	creator := &model.Participant{
		ConferenceID: conf.ID,
		UserID:       req.UserID,
		Status:       model.ParticipantStatusJoined,
		Role:         model.RoleOriginator,
		RSVPStatus:   model.RSVPAccepted,
		JoinedAt:     &now,
	}
	if err := s.confRepo.AddParticipantWithRole(ctx, creator); err != nil {
		s.logger.Error("failed to add creator as participant", zap.Error(err))
	}

	// Add selected participants (or all chat members if ParticipantIDs is empty)
	for _, participantID := range req.ParticipantIDs {
		if participantID == req.UserID {
			continue
		}
		p := &model.Participant{
			ConferenceID: conf.ID,
			UserID:       participantID,
			Status:       model.ParticipantStatusConnecting,
			Role:         model.RoleParticipant,
			RSVPStatus:   model.RSVPPending,
		}
		if err := s.confRepo.AddParticipantWithRole(ctx, p); err != nil {
			s.logger.Warn("failed to add participant", zap.String("userId", participantID.String()), zap.Error(err))
		}
	}

	// Get conference with participants
	conf, _ = s.confRepo.GetConferenceWithParticipants(ctx, conf.ID)

	// Publish event
	if err := s.eventPublisher.PublishConferenceCreated(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.created event", zap.Error(err))
	}

	s.logger.Info("ad-hoc call from chat created",
		zap.String("id", conf.ID.String()),
		zap.String("chatId", req.ChatID.String()),
		zap.Int("participantCount", len(req.ParticipantIDs)))

	return conf, nil
}

// CreateQuickAdHoc creates a quick ad-hoc call without a chat
func (s *voiceService) CreateQuickAdHoc(ctx context.Context, userID uuid.UUID, name string) (*model.Conference, error) {
	now := time.Now()
	fsName := fmt.Sprintf("adhoc_%s", uuid.New().String()[:8])

	if name == "" {
		name = fmt.Sprintf("Quick call %s", now.Format("2006-01-02 15:04"))
	}

	conf := &model.Conference{
		ID:             uuid.New(),
		Name:           name,
		FreeSwitchName: fsName,
		CreatedBy:      userID,
		Status:         model.ConferenceStatusActive,
		MaxMembers:     50,
		IsPrivate:      false,
		EventType:      model.EventTypeAdhoc,
		StartedAt:      &now,
	}

	// Create conference
	if err := s.confRepo.CreateScheduledConference(ctx, conf, nil); err != nil {
		return nil, fmt.Errorf("failed to create quick ad-hoc conference: %w", err)
	}

	// Create in FreeSWITCH
	if err := s.eslClient.CreateConference(ctx, fsName, "default"); err != nil {
		s.logger.Warn("failed to create conference in FreeSWITCH", zap.Error(err))
	}

	// Add creator as originator
	creator := &model.Participant{
		ConferenceID: conf.ID,
		UserID:       userID,
		Status:       model.ParticipantStatusJoined,
		Role:         model.RoleOriginator,
		RSVPStatus:   model.RSVPAccepted,
		JoinedAt:     &now,
	}
	if err := s.confRepo.AddParticipantWithRole(ctx, creator); err != nil {
		s.logger.Error("failed to add creator as participant", zap.Error(err))
	}

	// Publish event
	if err := s.eventPublisher.PublishConferenceCreated(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.created event", zap.Error(err))
	}

	s.logger.Info("quick ad-hoc call created",
		zap.String("id", conf.ID.String()),
		zap.String("name", name))

	return conf, nil
}

// UpdateRSVP updates a participant's RSVP status
func (s *voiceService) UpdateRSVP(ctx context.Context, req *model.UpdateRSVPRequest) (*model.Participant, error) {
	// Get participant
	participant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("participant not found: %w", err)
	}

	// Update RSVP status
	if err := s.confRepo.UpdateParticipantRSVP(ctx, participant.ID, req.RSVPStatus); err != nil {
		return nil, fmt.Errorf("failed to update RSVP: %w", err)
	}

	now := time.Now()
	participant.RSVPStatus = req.RSVPStatus
	participant.RSVPAt = &now

	// Publish event
	if err := s.eventPublisher.PublishRSVPUpdated(ctx, req.ConferenceID.String(), req.UserID.String(), req.RSVPStatus); err != nil {
		s.logger.Error("failed to publish rsvp_updated event", zap.Error(err))
	}

	s.logger.Info("RSVP updated",
		zap.String("conferenceId", req.ConferenceID.String()),
		zap.String("userId", req.UserID.String()),
		zap.String("status", string(req.RSVPStatus)))

	return participant, nil
}

// UpdateParticipantRole updates a participant's role with permission checks
func (s *voiceService) UpdateParticipantRole(ctx context.Context, req *model.UpdateParticipantRoleRequest) (*model.Participant, error) {
	// Get actor's participant record
	actorParticipant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.ActorUserID)
	if err != nil {
		return nil, fmt.Errorf("actor not found in conference: %w", err)
	}

	// Get target's participant record
	targetParticipant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.TargetUserID)
	if err != nil {
		return nil, fmt.Errorf("target participant not found: %w", err)
	}

	// Check permissions
	if !model.CanChangeRole(actorParticipant.Role, targetParticipant.Role, req.NewRole) {
		return nil, fmt.Errorf("not authorized to change role from %s to %s", targetParticipant.Role, req.NewRole)
	}

	oldRole := targetParticipant.Role

	// Update role
	if err := s.confRepo.UpdateParticipantRole(ctx, targetParticipant.ID, req.NewRole); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	targetParticipant.Role = req.NewRole

	// Publish event
	if err := s.eventPublisher.PublishParticipantRoleChanged(ctx, req.ConferenceID.String(), req.TargetUserID.String(), oldRole, req.NewRole); err != nil {
		s.logger.Error("failed to publish role_changed event", zap.Error(err))
	}

	s.logger.Info("participant role updated",
		zap.String("conferenceId", req.ConferenceID.String()),
		zap.String("targetUserId", req.TargetUserID.String()),
		zap.String("oldRole", string(oldRole)),
		zap.String("newRole", string(req.NewRole)))

	return targetParticipant, nil
}

// AddParticipants adds multiple participants to a conference
func (s *voiceService) AddParticipants(ctx context.Context, req *model.AddParticipantsRequest) error {
	// Get actor's participant record to check permissions
	actorParticipant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.ActorUserID)
	if err != nil {
		return fmt.Errorf("actor not found in conference: %w", err)
	}

	// Only originator and moderator can add participants
	if actorParticipant.Role != model.RoleOriginator && actorParticipant.Role != model.RoleModerator {
		return fmt.Errorf("not authorized to add participants")
	}

	// Get conference for scheduled_at (for reminders)
	conf, err := s.confRepo.GetConference(ctx, req.ConferenceID)
	if err != nil {
		return fmt.Errorf("conference not found: %w", err)
	}

	defaultRole := req.DefaultRole
	if defaultRole == "" {
		defaultRole = model.RoleParticipant
	}

	// Add each participant
	for _, userID := range req.UserIDs {
		p := &model.Participant{
			ConferenceID: req.ConferenceID,
			UserID:       userID,
			Status:       model.ParticipantStatusConnecting,
			Role:         defaultRole,
			RSVPStatus:   model.RSVPPending,
		}

		if err := s.confRepo.AddParticipantWithRole(ctx, p); err != nil {
			if err == repository.ErrAlreadyParticipant {
				continue // Skip if already exists
			}
			s.logger.Warn("failed to add participant", zap.String("userId", userID.String()), zap.Error(err))
			continue
		}

		// Create reminder for scheduled events
		if conf.ScheduledAt != nil {
			s.createReminder(ctx, req.ConferenceID, userID, *conf.ScheduledAt, 15)
		}

		// Publish event for each added participant
		if err := s.eventPublisher.PublishParticipantAdded(ctx, p); err != nil {
			s.logger.Error("failed to publish participant_added event", zap.Error(err))
		}
	}

	s.logger.Info("participants added to conference",
		zap.String("conferenceId", req.ConferenceID.String()),
		zap.Int("count", len(req.UserIDs)))

	return nil
}

// RemoveParticipant removes a participant from a conference
func (s *voiceService) RemoveParticipant(ctx context.Context, req *model.RemoveParticipantRequest) error {
	// Get actor's participant record
	actorParticipant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.ActorUserID)
	if err != nil {
		return fmt.Errorf("actor not found in conference: %w", err)
	}

	// Get target's participant record
	targetParticipant, err := s.confRepo.GetParticipant(ctx, req.ConferenceID, req.TargetUserID)
	if err != nil {
		return fmt.Errorf("target participant not found: %w", err)
	}

	// Check permissions - originator can remove anyone, moderator can remove participant/speaker/assistant
	canRemove := false
	if actorParticipant.Role == model.RoleOriginator {
		canRemove = true
	} else if actorParticipant.Role == model.RoleModerator {
		if targetParticipant.Role == model.RoleParticipant ||
			targetParticipant.Role == model.RoleSpeaker ||
			targetParticipant.Role == model.RoleAssistant {
			canRemove = true
		}
	}

	if !canRemove {
		return fmt.Errorf("not authorized to remove this participant")
	}

	// Remove participant
	if err := s.confRepo.RemoveParticipant(ctx, req.ConferenceID, req.TargetUserID); err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	// Publish event
	if err := s.eventPublisher.PublishParticipantRemoved(ctx, req.ConferenceID.String(), req.TargetUserID.String()); err != nil {
		s.logger.Error("failed to publish participant_removed event", zap.Error(err))
	}

	s.logger.Info("participant removed from conference",
		zap.String("conferenceId", req.ConferenceID.String()),
		zap.String("targetUserId", req.TargetUserID.String()))

	return nil
}

// ListScheduledConferences lists scheduled conferences for a user
func (s *voiceService) ListScheduledConferences(ctx context.Context, userID uuid.UUID, upcomingOnly bool, limit, offset int) ([]*model.Conference, int, error) {
	return s.confRepo.ListScheduledConferences(ctx, userID, upcomingOnly, limit, offset)
}

// GetChatConferences gets conferences for a specific chat
func (s *voiceService) GetChatConferences(ctx context.Context, chatID uuid.UUID, upcomingOnly bool) ([]*model.Conference, error) {
	return s.confRepo.GetChatConferences(ctx, chatID, upcomingOnly)
}

// CancelConference cancels a scheduled conference
func (s *voiceService) CancelConference(ctx context.Context, confID, userID uuid.UUID, cancelSeries bool) error {
	// Get conference
	conf, err := s.confRepo.GetConference(ctx, confID)
	if err != nil {
		return fmt.Errorf("conference not found: %w", err)
	}

	// Get user's participant record
	participant, err := s.confRepo.GetParticipant(ctx, confID, userID)
	if err != nil {
		return fmt.Errorf("user not found in conference: %w", err)
	}

	// Only originator can cancel
	if participant.Role != model.RoleOriginator {
		return fmt.Errorf("not authorized to cancel conference")
	}

	// Update status to cancelled
	if err := s.confRepo.UpdateConferenceStatus(ctx, confID, model.ConferenceStatusCancelled, nil); err != nil {
		return fmt.Errorf("failed to cancel conference: %w", err)
	}

	// If recurring and cancelSeries, cancel all in series
	if cancelSeries && conf.SeriesID != nil {
		// TODO: Cancel all conferences in series
		s.logger.Info("cancelling series", zap.String("seriesId", conf.SeriesID.String()))
	}

	// Publish event
	if err := s.eventPublisher.PublishConferenceCancelled(ctx, conf); err != nil {
		s.logger.Error("failed to publish conference.cancelled event", zap.Error(err))
	}

	s.logger.Info("conference cancelled",
		zap.String("conferenceId", confID.String()),
		zap.Bool("cancelSeries", cancelSeries))

	return nil
}

// createReminder creates a reminder for a participant
func (s *voiceService) createReminder(ctx context.Context, confID, userID uuid.UUID, scheduledAt time.Time, minutesBefore int) {
	remindAt := scheduledAt.Add(-time.Duration(minutesBefore) * time.Minute)

	reminder := &model.ConferenceReminder{
		ConferenceID:  confID,
		UserID:        userID,
		RemindAt:      remindAt,
		MinutesBefore: minutesBefore,
	}

	if err := s.confRepo.CreateReminder(ctx, reminder); err != nil {
		s.logger.Warn("failed to create reminder",
			zap.String("conferenceId", confID.String()),
			zap.String("userId", userID.String()),
			zap.Error(err))
	}
}
