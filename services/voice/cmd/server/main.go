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

	pb "github.com/icegreg/chat-smpl/proto/voice"
	"github.com/icegreg/chat-smpl/services/voice/internal/config"
	"github.com/icegreg/chat-smpl/services/voice/internal/esl"
	"github.com/icegreg/chat-smpl/services/voice/internal/events"
	voicegrpc "github.com/icegreg/chat-smpl/services/voice/internal/grpc"
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

	// Initialize service
	voiceService := service.NewVoiceService(
		cfg,
		eslClient,
		confRepo,
		callRepo,
		eventPublisher,
		logger,
	)

	// Initialize and start scheduler for reminders and recurring events
	reminderScheduler := scheduler.NewScheduler(confRepo, eventPublisher, logger)
	reminderScheduler.Start(ctx)

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

	// FreeSWITCH webhook endpoint for events
	httpMux.HandleFunc("/webhook/freeswitch", func(w http.ResponseWriter, r *http.Request) {
		// Handle FreeSWITCH HTTP events if needed
		w.WriteHeader(http.StatusOK)
	})

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
		handleFreeSwitchEvent(ctx, event, voiceService, confRepo, callRepo, eventPublisher, logger)
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
		// Call ended
		uuid := event.GetHeader("Unique-ID")
		cause := event.GetHeader("Hangup-Cause")
		logger.Info("channel hangup", zap.String("uuid", uuid), zap.String("cause", cause))

	case "CONFERENCE_DATA":
		// Conference data event
		confName := event.GetHeader("Conference-Name")
		action := event.GetHeader("Action")
		logger.Info("conference event", zap.String("confName", confName), zap.String("action", action))

	case "CONFERENCE_MEMBER_FLAGS":
		// Member flags changed (mute/deaf/etc)
		confName := event.GetHeader("Conference-Name")
		memberID := event.GetHeader("Member-ID")
		logger.Debug("conference member flags changed",
			zap.String("confName", confName),
			zap.String("memberID", memberID))
	}
}
