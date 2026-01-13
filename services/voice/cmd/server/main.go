package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/icegreg/chat-smpl/pkg/metrics"
	pb "github.com/icegreg/chat-smpl/proto/voice"
	"github.com/icegreg/chat-smpl/services/voice/internal/chatclient"
	"github.com/icegreg/chat-smpl/services/voice/internal/config"
	"github.com/icegreg/chat-smpl/services/voice/internal/esl"
	"github.com/icegreg/chat-smpl/services/voice/internal/events"
	"github.com/icegreg/chat-smpl/services/voice/internal/fsdirectory"
	voicegrpc "github.com/icegreg/chat-smpl/services/voice/internal/grpc"
	"github.com/icegreg/chat-smpl/services/voice/internal/model"
	"github.com/icegreg/chat-smpl/services/voice/internal/repository"
	"github.com/icegreg/chat-smpl/services/voice/internal/scheduler"
	"github.com/icegreg/chat-smpl/services/voice/internal/service"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("starting voice service",
		zap.String("grpcAddr", cfg.GRPCAddr),
		zap.Int("httpPort", cfg.HTTPPort))

	// Connect to PostgreSQL
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal("failed to ping database", zap.Error(err))
	}
	logger.Info("connected to database")

	// Initialize ESL client
	eslClient := esl.NewClient(
		cfg.FreeSWITCH.ESLHost,
		cfg.FreeSWITCH.ESLPort,
		cfg.FreeSWITCH.ESLPassword,
		logger,
	)

	// Connect to FreeSWITCH (non-blocking, will reconnect)
	go func() {
		for i := 0; i < 10; i++ {
			if err := eslClient.Connect(ctx); err != nil {
				logger.Warn("failed to connect to FreeSWITCH, will retry",
					zap.Error(err),
					zap.Int("attempt", i+1))
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			logger.Info("connected to FreeSWITCH ESL")

			// Subscribe to events
			if err := eslClient.SubscribeEvents(
				"CHANNEL_CREATE",
				"CHANNEL_ANSWER",
				"CHANNEL_HANGUP",
				"CONFERENCE_DATA",
				"CONFERENCE_MEMBER_FLAGS",
			); err != nil {
				logger.Warn("failed to subscribe to events", zap.Error(err))
			}
			break
		}
	}()

	// Initialize RabbitMQ publisher
	eventPublisher, err := events.NewPublisher(cfg.RabbitMQURL, logger)
	if err != nil {
		logger.Fatal("failed to create event publisher", zap.Error(err))
	}
	defer eventPublisher.Close()

	// Initialize repositories
	confRepo := repository.NewConferenceRepository(pool)
	callRepo := repository.NewCallRepository(pool)

	// Initialize chat client for system messages
	var chatClientPtr *chatclient.ChatClient
	if cfg.ChatServiceAddr != "" {
		chatClient, err := chatclient.NewChatClient(cfg.ChatServiceAddr)
		if err != nil {
			logger.Warn("failed to create chat client, system messages will be disabled",
				zap.Error(err),
				zap.String("chatServiceAddr", cfg.ChatServiceAddr))
		} else {
			chatClientPtr = chatClient
			defer chatClient.Close()
			logger.Info("connected to chat service", zap.String("addr", cfg.ChatServiceAddr))
		}
	}

	// Initialize service
	voiceService := service.NewVoiceService(
		cfg,
		pool,
		eslClient,
		confRepo,
		callRepo,
		eventPublisher,
		chatClientPtr,
		logger,
	)

	// Initialize and start scheduler for reminders and recurring events
	reminderScheduler := scheduler.NewScheduler(confRepo, eventPublisher, logger)
	reminderScheduler.Start(ctx)

	// Start cleanup worker for stale conferences
	go startCleanupWorker(ctx, confRepo, eventPublisher, cfg, logger)
	go startEmptyConferenceMonitor(ctx, confRepo, eventPublisher, eslClient, cfg, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register voice service
	voiceServer := voicegrpc.NewServer(voiceService, logger)
	pb.RegisterVoiceServiceServer(grpcServer, voiceServer)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for grpcurl/grpcui
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Fatal("failed to listen for gRPC", zap.Error(err))
	}

	go func() {
		logger.Info("gRPC server starting", zap.String("addr", cfg.GRPCAddr))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server failed", zap.Error(err))
		}
	}()

	// Start HTTP server for health checks and webhooks
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	httpMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("database not ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics endpoint
	httpMux.Handle("/metrics", metrics.Handler())

	// FreeSWITCH webhook endpoint for events
	httpMux.HandleFunc("/webhook/freeswitch", func(w http.ResponseWriter, r *http.Request) {
		// Handle FreeSWITCH HTTP events if needed
		w.WriteHeader(http.StatusOK)
	})

	// FreeSWITCH directory handler for mod_xml_curl
	fsDirectoryHandler := fsdirectory.NewHandler(pool, cfg.Verto.Domain, logger)
	httpMux.Handle("/fs/directory", fsDirectoryHandler)

	httpAddr := fmt.Sprintf(":%d", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: httpMux,
	}

	go func() {
		logger.Info("HTTP server starting", zap.String("addr", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	// Setup ESL event handler
	eslClient.OnEvent(func(event *esl.Event) {
		handleFreeSwitchEvent(ctx, event, voiceService, confRepo, callRepo, eventPublisher, eslClient, logger)
	})

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down voice service")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	httpServer.Shutdown(shutdownCtx)
	eslClient.Close()

	logger.Info("voice service stopped")
}

// handleFreeSwitchEvent processes events from FreeSWITCH
func handleFreeSwitchEvent(
	ctx context.Context,
	event *esl.Event,
	voiceService service.VoiceService,
	confRepo repository.ConferenceRepository,
	callRepo repository.CallRepository,
	eventPublisher events.Publisher,
	eslClient esl.Client,
	logger *zap.Logger,
) {
	if event == nil {
		return
	}

	eventName := event.GetHeader("Event-Name")
	logger.Debug("received FreeSWITCH event", zap.String("eventName", eventName))

	switch eventName {
	case "CHANNEL_CREATE":
		// New channel created
		uuid := event.GetHeader("Unique-ID")
		logger.Info("channel created", zap.String("uuid", uuid))

	case "CHANNEL_ANSWER":
		// Call answered
		uuid := event.GetHeader("Unique-ID")
		logger.Info("channel answered", zap.String("uuid", uuid))

	case "CHANNEL_HANGUP":
		// Call ended - handle participant leaving
		channelUUID := event.GetHeader("Unique-ID")
		cause := event.GetHeader("Hangup-Cause")
		logger.Info("channel hangup", zap.String("uuid", channelUUID), zap.String("cause", cause))

		// Find participant by channel UUID
		participant, err := confRepo.GetParticipantByChannelUUID(ctx, channelUUID)
		if err != nil {
			logger.Debug("participant not found for channel UUID",
				zap.String("channelUUID", channelUUID),
				zap.Error(err))
			return
		}

		// Update participant status to left
		now := time.Now()
		if err := confRepo.UpdateParticipantStatus(ctx, participant.ID, model.ParticipantStatusLeft, &now); err != nil {
			logger.Error("failed to update participant status on hangup",
				zap.String("participantID", participant.ID.String()),
				zap.Error(err))
			return
		}

		logger.Info("participant left due to channel hangup",
			zap.String("participantID", participant.ID.String()),
			zap.String("conferenceID", participant.ConferenceID.String()),
			zap.String("userID", participant.UserID.String()))

		// Publish participant left event
		if eventPublisher != nil {
			// Get conference to retrieve ChatID
			chatID := ""
			conf, err := confRepo.GetConference(ctx, participant.ConferenceID)
			if err == nil && conf.ChatID != nil {
				chatID = conf.ChatID.String()
			}
			_ = eventPublisher.PublishParticipantLeft(ctx, participant, chatID)
		}

		// Check if conference should end
		// Criteria for ending:
		// 1. No active participants in DB (activeCount == 0)
		// 2. OR no real WebRTC participants in FreeSWITCH
		shouldEnd := false
		endReason := ""

		// Check DB participant count
		activeCount, err := confRepo.GetActiveParticipantCount(ctx, participant.ConferenceID)
		if err != nil {
			logger.Error("failed to get active participant count", zap.Error(err))
			return
		}

		if activeCount == 0 {
			shouldEnd = true
			endReason = "all participants left (DB)"
		} else {
			// Check real FreeSWITCH participants
			conf, err := confRepo.GetConference(ctx, participant.ConferenceID)
			if err != nil {
				logger.Debug("failed to get conference for FS check", zap.Error(err))
			} else if eslClient != nil && eslClient.IsConnected() {
				fsMembers, err := eslClient.GetConferenceMembers(ctx, conf.FreeSwitchName)
				if err != nil {
					logger.Debug("failed to get FreeSWITCH conference members", zap.Error(err))
				} else if len(fsMembers) == 0 {
					shouldEnd = true
					endReason = "no WebRTC participants in FreeSWITCH"
				}
			}
		}

		if shouldEnd {
			logger.Info("ending conference due to no participants",
				zap.String("conferenceID", participant.ConferenceID.String()),
				zap.String("reason", endReason),
				zap.Int("dbActiveCount", activeCount))

			// Get conference object for event publishing
			conf, err := confRepo.GetConference(ctx, participant.ConferenceID)
			if err != nil {
				logger.Error("failed to get conference", zap.Error(err))
				return
			}

			// End conference
			endTime := time.Now()
			if err := confRepo.UpdateConferenceStatus(ctx, participant.ConferenceID, model.ConferenceStatusEnded, &endTime); err != nil {
				logger.Error("failed to end conference", zap.Error(err))
				return
			}

			// Update conference object for event
			conf.Status = model.ConferenceStatusEnded
			conf.EndedAt = &endTime

			// Publish conference ended event
			if eventPublisher != nil {
				_ = eventPublisher.PublishConferenceEnded(ctx, conf)
			}
		}

	case "CONFERENCE_DATA":
		// Conference data event - handle member add/del
		confName := event.GetHeader("Conference-Name")
		action := event.GetHeader("Action")
		memberID := event.GetHeader("Member-ID")
		channelUUID := event.GetHeader("Unique-ID")

		logger.Info("conference event",
			zap.String("confName", confName),
			zap.String("action", action),
			zap.String("memberID", memberID),
			zap.String("channelUUID", channelUUID))

		if action == "add-member" && channelUUID != "" {
			// Find conference by FreeSWITCH name
			conf, err := confRepo.GetConferenceByFSName(ctx, confName)
			if err != nil {
				logger.Debug("conference not found", zap.String("confName", confName), zap.Error(err))
				return
			}

			// Try to find participant by channel UUID or by conference
			// First, try all participants to find one without channel_uuid set
			participants, err := confRepo.ListParticipants(ctx, conf.ID)
			if err != nil {
				logger.Error("failed to list participants", zap.Error(err))
				return
			}

			// Find participant without channel_uuid (just joined)
			for _, p := range participants {
				if p.ChannelUUID == nil || *p.ChannelUUID == "" {
					// Set channel UUID and FS member ID
					if err := confRepo.SetParticipantChannelUUID(ctx, p.ID, channelUUID); err != nil {
						logger.Error("failed to set channel UUID", zap.Error(err))
						continue
					}
					if memberID != "" {
						if err := confRepo.SetParticipantFSMemberID(ctx, p.ID, memberID); err != nil {
							logger.Error("failed to set FS member ID", zap.Error(err))
						}
					}
					logger.Info("associated participant with FreeSWITCH member",
						zap.String("participantID", p.ID.String()),
						zap.String("channelUUID", channelUUID),
						zap.String("memberID", memberID))
					break
				}
			}
		}

	case "CONFERENCE_MEMBER_FLAGS":
		// Member flags changed (mute/deaf/etc)
		confName := event.GetHeader("Conference-Name")
		memberID := event.GetHeader("Member-ID")
		logger.Debug("conference member flags changed",
			zap.String("confName", confName),
			zap.String("memberID", memberID))
	}
}

// startCleanupWorker runs a periodic cleanup of stale conferences
func startCleanupWorker(
	ctx context.Context,
	confRepo repository.ConferenceRepository,
	eventPublisher events.Publisher,
	cfg *config.Config,
	logger *zap.Logger,
) {
	// Run cleanup every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Max age for conferences - 24 hours
	maxAge := 24 * time.Hour

	logger.Info("cleanup worker started",
		zap.Duration("interval", 5*time.Minute),
		zap.Duration("maxAge", maxAge))

	// Run initial cleanup
	runCleanup(ctx, confRepo, maxAge, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("cleanup worker stopped")
			return
		case <-ticker.C:
			runCleanup(ctx, confRepo, maxAge, logger)
		}
	}
}

// startEmptyConferenceMonitor checks for empty conferences and ends them after timeout
func startEmptyConferenceMonitor(
	ctx context.Context,
	confRepo repository.ConferenceRepository,
	eventPublisher events.Publisher,
	eslClient esl.Client,
	cfg *config.Config,
	logger *zap.Logger,
) {
	// Check every 30 seconds
	checkInterval := 30 * time.Second
	emptyTimeout := time.Duration(cfg.EmptyConferenceTimeout) * time.Second

	logger.Info("empty conference monitor started",
		zap.Duration("checkInterval", checkInterval),
		zap.Duration("emptyTimeout", emptyTimeout))

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("empty conference monitor stopped")
			return
		case <-ticker.C:
			checkEmptyConferences(ctx, confRepo, eventPublisher, eslClient, emptyTimeout, logger)
		}
	}
}

// checkEmptyConferences finds and ends empty conferences that exceed timeout
func checkEmptyConferences(
	ctx context.Context,
	confRepo repository.ConferenceRepository,
	eventPublisher events.Publisher,
	eslClient esl.Client,
	emptyTimeout time.Duration,
	logger *zap.Logger,
) {
	// First, cleanup participants stuck in 'connecting' status (e.g., users who clicked Call All but never connected)
	connectingTimeout := 2 * time.Minute // 2 minutes to connect
	cleanedParticipants, err := confRepo.CleanupStaleConnectingParticipants(ctx, connectingTimeout)
	if err != nil {
		logger.Error("failed to cleanup stale connecting participants", zap.Error(err))
	} else if cleanedParticipants > 0 {
		logger.Info("cleaned up stale connecting participants",
			zap.Int("count", cleanedParticipants),
			zap.Duration("timeout", connectingTimeout))
	}

	// Get all active conferences
	activeConfs, err := confRepo.ListAllActiveConferences(ctx)
	if err != nil {
		logger.Error("failed to list active conferences", zap.Error(err))
		return
	}

	now := time.Now()
	for _, conf := range activeConfs {
		// Check if conference has been active for more than the empty timeout
		if conf.StartedAt == nil {
			continue // Skip conferences that haven't started yet
		}

		// Check if conference should end
		// Criteria:
		// 1. No active participants in DB
		// 2. OR no real WebRTC participants in FreeSWITCH
		// 3. AND empty for longer than timeout
		shouldEnd := false
		endReason := ""

		// Check DB participant count
		activeCount, err := confRepo.GetActiveParticipantCount(ctx, conf.ID)
		if err != nil {
			logger.Error("failed to get active participant count",
				zap.String("conferenceID", conf.ID.String()),
				zap.Error(err))
			continue
		}

		// Calculate how long conference has been running
		runningDuration := now.Sub(*conf.StartedAt)

		if activeCount == 0 {
			// No participants in DB - check timeout
			if runningDuration > emptyTimeout {
				shouldEnd = true
				endReason = "no DB participants, timeout exceeded"
			}
		} else {
			// Have participants in DB - check FreeSWITCH reality
			if eslClient != nil && eslClient.IsConnected() {
				fsMembers, err := eslClient.GetConferenceMembers(ctx, conf.FreeSwitchName)
				if err != nil {
					logger.Debug("failed to get FreeSWITCH members for conference",
						zap.String("conferenceID", conf.ID.String()),
						zap.String("fsName", conf.FreeSwitchName),
						zap.Error(err))
				} else if len(fsMembers) == 0 {
					// No real WebRTC participants - check timeout
					if runningDuration > emptyTimeout {
						shouldEnd = true
						endReason = "no WebRTC participants in FreeSWITCH, timeout exceeded"
					}
				}
			}
		}

		if shouldEnd {
			logger.Info("ending empty conference due to timeout",
				zap.String("conferenceID", conf.ID.String()),
				zap.String("name", conf.Name),
				zap.String("reason", endReason),
				zap.Int("dbActiveCount", activeCount),
				zap.Duration("runningFor", runningDuration))

			endTime := now
			if err := confRepo.UpdateConferenceStatus(ctx, conf.ID, model.ConferenceStatusEnded, &endTime); err != nil {
				logger.Error("failed to end empty conference",
					zap.String("conferenceID", conf.ID.String()),
					zap.Error(err))
				continue
			}

			// Update conference object for event
			conf.Status = model.ConferenceStatusEnded
			conf.EndedAt = &endTime

			// Publish conference ended event
			if eventPublisher != nil {
				_ = eventPublisher.PublishConferenceEnded(ctx, conf)
			}
		}
	}
}

func runCleanup(ctx context.Context, confRepo repository.ConferenceRepository, maxAge time.Duration, logger *zap.Logger) {
	cleaned, err := confRepo.CleanupStaleConferences(ctx, maxAge)
	if err != nil {
		logger.Error("failed to cleanup stale conferences", zap.Error(err))
		return
	}

	if cleaned > 0 {
		logger.Info("cleaned up stale conferences", zap.Int("count", cleaned))
	}
}
