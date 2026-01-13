package checker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/icegreg/chat-smpl/services/health/internal/centrifugo"
	"github.com/icegreg/chat-smpl/services/health/internal/config"
	chatpb "github.com/icegreg/chat-smpl/proto/chat"
	voicepb "github.com/icegreg/chat-smpl/proto/voice"
)

type HealthStatus string

const (
	StatusOK       HealthStatus = "OK"
	StatusDegraded HealthStatus = "DEGRADED"
	StatusDown     HealthStatus = "DOWN"
)

type HealthCheckResult struct {
	Status               HealthStatus `json:"status"`
	LastCheckTime        time.Time    `json:"last_check_time"`
	TotalRoundtripMs     int64        `json:"total_roundtrip_ms"`
	APItoChatServiceMs   *int64       `json:"api_to_chat_service_ms,omitempty"`
	ErrorMessage         string       `json:"error_message,omitempty"`
	FailedStage          string       `json:"failed_stage,omitempty"`
	ConsecutiveFailures  int          `json:"consecutive_failures"`
	CentrifugoConnected  bool         `json:"centrifugo_connected"`

	// Voice metrics
	VoiceCheckEnabled    bool   `json:"voice_check_enabled"`
	VoiceStatus          string `json:"voice_status,omitempty"`           // OK, ERROR, DISABLED
	CreateConferenceMs   *int64 `json:"create_conference_ms,omitempty"`
	AddParticipantsMs    *int64 `json:"add_participants_ms,omitempty"`
	EndConferenceMs      *int64 `json:"end_conference_ms,omitempty"`
	VoiceTotalMs         *int64 `json:"voice_total_ms,omitempty"`
	VoiceErrorMessage    string `json:"voice_error_message,omitempty"`
}

type Checker struct {
	config        *config.Config
	chatClient    chatpb.ChatServiceClient
	voiceClient   voicepb.VoiceServiceClient
	chatGrpcConn  *grpc.ClientConn
	voiceGrpcConn *grpc.ClientConn
	subscriber    *centrifugo.Subscriber
	log           *slog.Logger

	mu               sync.RWMutex
	lastResult       *HealthCheckResult
	consecutiveFails int
	stopCh           chan struct{}
}

func NewChecker(cfg *config.Config, log *slog.Logger) (*Checker, error) {
	// gRPC dial options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Connect to chat-service gRPC
	chatConn, err := grpc.NewClient(cfg.ChatServiceAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to chat-service: %w", err)
	}

	// Connect to voice-service gRPC (if enabled)
	var voiceConn *grpc.ClientConn
	var voiceClient voicepb.VoiceServiceClient
	if cfg.VoiceCheckEnabled {
		voiceConn, err = grpc.NewClient(cfg.VoiceServiceAddr, opts...)
		if err != nil {
			log.Warn("failed to connect to voice-service, voice checks disabled", "error", err)
		} else {
			voiceClient = voicepb.NewVoiceServiceClient(voiceConn)
		}
	}

	// Create Centrifugo subscriber
	subscriber := centrifugo.NewSubscriber(
		cfg.CentrifugoWSURL,
		cfg.CentrifugoSecret,
		cfg.SystemUserID,
		log,
	)

	return &Checker{
		config:        cfg,
		chatClient:    chatpb.NewChatServiceClient(chatConn),
		voiceClient:   voiceClient,
		chatGrpcConn:  chatConn,
		voiceGrpcConn: voiceConn,
		subscriber:    subscriber,
		log:           log,
		stopCh:        make(chan struct{}),
		lastResult: &HealthCheckResult{
			Status:            StatusDown,
			LastCheckTime:     time.Now(),
			ErrorMessage:      "not yet checked",
			VoiceCheckEnabled: cfg.VoiceCheckEnabled,
		},
	}, nil
}

func (c *Checker) Start(ctx context.Context) error {
	// Connect to Centrifugo first
	if err := c.subscriber.Connect(ctx); err != nil {
		c.log.Warn("failed to connect to Centrifugo, will retry", "error", err)
		// Don't fail startup, the checker will handle this
	}

	go c.runLoop(ctx)
	return nil
}

func (c *Checker) runLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.CheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	c.runCheck(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.runCheck(ctx)
		}
	}
}

func (c *Checker) runCheck(ctx context.Context) {
	result := c.performCheck(ctx)

	// Run voice check if enabled and chat check succeeded
	if c.config.VoiceCheckEnabled && c.voiceClient != nil {
		c.performVoiceCheck(ctx, result)
	} else if c.config.VoiceCheckEnabled {
		result.VoiceCheckEnabled = true
		result.VoiceStatus = "DISABLED"
		result.VoiceErrorMessage = "voice client not connected"
	}

	c.mu.Lock()
	c.lastResult = result
	c.mu.Unlock()

	logFields := []any{
		"status", result.Status,
		"roundtrip_ms", result.TotalRoundtripMs,
		"consecutive_failures", result.ConsecutiveFailures,
	}
	if result.VoiceCheckEnabled {
		logFields = append(logFields,
			"voice_status", result.VoiceStatus,
		)
		if result.VoiceTotalMs != nil {
			logFields = append(logFields, "voice_total_ms", *result.VoiceTotalMs)
		}
	}
	c.log.Info("health check completed", logFields...)
}

func (c *Checker) performCheck(ctx context.Context) *HealthCheckResult {
	startTime := time.Now()
	result := &HealthCheckResult{
		LastCheckTime:       startTime,
		CentrifugoConnected: c.subscriber.IsConnected(),
	}

	// Check if Centrifugo is connected
	if !c.subscriber.IsConnected() {
		// Try to reconnect
		if err := c.subscriber.Connect(ctx); err != nil {
			c.consecutiveFails++
			result.Status = StatusDown
			result.ErrorMessage = fmt.Sprintf("Centrifugo not connected: %v", err)
			result.FailedStage = "centrifugo_connect"
			result.ConsecutiveFailures = c.consecutiveFails
			return result
		}
		result.CentrifugoConnected = true
	}

	// Generate unique message content with timestamp for identification
	messageID := uuid.New().String()
	content := fmt.Sprintf("__health_check__%s__%d", messageID, startTime.UnixNano())

	// Send message via gRPC
	grpcStart := time.Now()
	msg, err := c.chatClient.SendMessage(ctx, &chatpb.SendMessageRequest{
		ChatId:   c.config.SystemChatID,
		SenderId: c.config.SystemUserID,
		Content:  content,
	})
	grpcDuration := time.Since(grpcStart).Milliseconds()
	result.APItoChatServiceMs = &grpcDuration

	if err != nil {
		c.consecutiveFails++
		result.Status = StatusDown
		result.ErrorMessage = fmt.Sprintf("gRPC SendMessage failed: %v", err)
		result.FailedStage = "grpc_send"
		result.ConsecutiveFailures = c.consecutiveFails
		result.TotalRoundtripMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Wait for message to arrive via Centrifugo
	receivedMsg, err := c.subscriber.WaitForMessage(ctx, msg.Id, c.config.MessageTimeout)
	if err != nil {
		c.consecutiveFails++
		result.Status = StatusDown
		result.ErrorMessage = fmt.Sprintf("Centrifugo message not received: %v", err)
		result.FailedStage = "centrifugo_receive"
		result.ConsecutiveFailures = c.consecutiveFails
		result.TotalRoundtripMs = time.Since(startTime).Milliseconds()
		return result
	}

	// Calculate total roundtrip
	result.TotalRoundtripMs = receivedMsg.ReceivedAt.Sub(startTime).Milliseconds()

	// Determine status based on thresholds
	c.consecutiveFails = 0
	result.ConsecutiveFailures = 0

	if result.TotalRoundtripMs > c.config.DownThresholdMs {
		result.Status = StatusDown
		result.ErrorMessage = fmt.Sprintf("roundtrip too slow: %dms > %dms", result.TotalRoundtripMs, c.config.DownThresholdMs)
	} else if result.TotalRoundtripMs > c.config.DegradedThresholdMs {
		result.Status = StatusDegraded
	} else {
		result.Status = StatusOK
	}

	return result
}

func (c *Checker) GetLastResult() *HealthCheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastResult == nil {
		return &HealthCheckResult{
			Status:        StatusDown,
			LastCheckTime: time.Now(),
			ErrorMessage:  "no check performed yet",
		}
	}

	// Return a copy
	result := *c.lastResult
	return &result
}

func (c *Checker) Stop() error {
	close(c.stopCh)
	if err := c.subscriber.Close(); err != nil {
		c.log.Error("failed to close subscriber", "error", err)
	}
	if c.voiceGrpcConn != nil {
		if err := c.voiceGrpcConn.Close(); err != nil {
			c.log.Error("failed to close voice grpc conn", "error", err)
		}
	}
	return c.chatGrpcConn.Close()
}

// performVoiceCheck runs the voice service health check
// It creates a conference, adds participants (join), then ends it
func (c *Checker) performVoiceCheck(ctx context.Context, result *HealthCheckResult) {
	result.VoiceCheckEnabled = true
	voiceStart := time.Now()

	// 1. Create conference
	createStart := time.Now()
	confName := fmt.Sprintf("__health_check__%d", voiceStart.UnixNano())
	conf, err := c.voiceClient.CreateConference(ctx, &voicepb.CreateConferenceRequest{
		Name:            confName,
		CreatedBy:       c.config.SystemUserID,
		MaxMembers:      10,
		IsPrivate:       true,
		EnableRecording: false,
	})
	createDuration := time.Since(createStart).Milliseconds()
	result.CreateConferenceMs = &createDuration

	if err != nil {
		result.VoiceStatus = "ERROR"
		result.VoiceErrorMessage = fmt.Sprintf("CreateConference failed: %v", err)
		totalMs := time.Since(voiceStart).Milliseconds()
		result.VoiceTotalMs = &totalMs
		return
	}

	// 2. Join conference (creator joins)
	addStart := time.Now()
	_, err = c.voiceClient.JoinConference(ctx, &voicepb.JoinConferenceRequest{
		ConferenceId: conf.Id,
		UserId:       c.config.SystemUserID,
		Muted:        true,
	})
	if err != nil {
		result.VoiceStatus = "ERROR"
		result.VoiceErrorMessage = fmt.Sprintf("JoinConference (creator) failed: %v", err)
		totalMs := time.Since(voiceStart).Milliseconds()
		result.VoiceTotalMs = &totalMs
		// Try to cleanup
		_, _ = c.voiceClient.EndConference(ctx, &voicepb.EndConferenceRequest{
			ConferenceId: conf.Id,
			UserId:       c.config.SystemUserID,
		})
		return
	}

	// Join 2 additional participants
	_, err = c.voiceClient.JoinConference(ctx, &voicepb.JoinConferenceRequest{
		ConferenceId: conf.Id,
		UserId:       c.config.SystemUser2ID,
		Muted:        true,
	})
	if err != nil {
		result.VoiceStatus = "ERROR"
		result.VoiceErrorMessage = fmt.Sprintf("JoinConference (user2) failed: %v", err)
		totalMs := time.Since(voiceStart).Milliseconds()
		result.VoiceTotalMs = &totalMs
		_, _ = c.voiceClient.EndConference(ctx, &voicepb.EndConferenceRequest{
			ConferenceId: conf.Id,
			UserId:       c.config.SystemUserID,
		})
		return
	}

	_, err = c.voiceClient.JoinConference(ctx, &voicepb.JoinConferenceRequest{
		ConferenceId: conf.Id,
		UserId:       c.config.SystemUser3ID,
		Muted:        true,
	})
	addDuration := time.Since(addStart).Milliseconds()
	result.AddParticipantsMs = &addDuration

	if err != nil {
		result.VoiceStatus = "ERROR"
		result.VoiceErrorMessage = fmt.Sprintf("JoinConference (user3) failed: %v", err)
		totalMs := time.Since(voiceStart).Milliseconds()
		result.VoiceTotalMs = &totalMs
		_, _ = c.voiceClient.EndConference(ctx, &voicepb.EndConferenceRequest{
			ConferenceId: conf.Id,
			UserId:       c.config.SystemUserID,
		})
		return
	}

	// 3. End conference (cleanup)
	endStart := time.Now()
	_, err = c.voiceClient.EndConference(ctx, &voicepb.EndConferenceRequest{
		ConferenceId: conf.Id,
		UserId:       c.config.SystemUserID,
	})
	endDuration := time.Since(endStart).Milliseconds()
	result.EndConferenceMs = &endDuration

	if err != nil {
		result.VoiceStatus = "ERROR"
		result.VoiceErrorMessage = fmt.Sprintf("EndConference failed: %v", err)
		totalMs := time.Since(voiceStart).Milliseconds()
		result.VoiceTotalMs = &totalMs
		return
	}

	// Success
	totalMs := time.Since(voiceStart).Milliseconds()
	result.VoiceTotalMs = &totalMs
	result.VoiceStatus = "OK"
}
