package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/icegreg/chat-smpl/pkg/metrics"
	"github.com/icegreg/chat-smpl/services/admin/internal/client"
	"github.com/icegreg/chat-smpl/services/admin/internal/handler"
	"github.com/icegreg/chat-smpl/services/admin/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// Get config from environment
	dbURL := getEnv("DATABASE_URL", "postgres://chatuser:chatpass@postgres:5432/chatdb?sslmode=disable")
	voiceServiceAddr := getEnv("VOICE_SERVICE_ADDR", "voice-service:50053")
	httpPort := getEnv("HTTP_PORT", "8085")

	// Connect to database
	logger.Info("connecting to database", zap.String("url", dbURL))
	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Initialize voice service client
	logger.Info("connecting to voice service", zap.String("addr", voiceServiceAddr))
	voiceClient, err := client.NewVoiceClient(voiceServiceAddr, logger)
	if err != nil {
		logger.Fatal("failed to create voice client", zap.Error(err))
	}
	defer voiceClient.Close()

	// Initialize services
	conferenceService := service.NewConferenceService(dbPool, voiceClient, logger)
	serviceMonitor := service.NewServiceMonitor(logger)

	// Initialize handlers
	conferenceHandler := handler.NewConferenceHandler(conferenceService, logger)
	serviceHandler := handler.NewServiceHandler(serviceMonitor, logger)

	// Setup HTTP router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Prometheus metrics
	r.Handle("/metrics", metrics.Handler())

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Conference endpoints
		r.Route("/conferences", func(r chi.Router) {
			r.Get("/", conferenceHandler.ListConferences)
			r.Get("/{id}", conferenceHandler.GetConference)
			r.Get("/{id}/participants", conferenceHandler.ListParticipants)
			r.Post("/{id}/end", conferenceHandler.EndConference)
		})

		// Service endpoints
		r.Route("/services", func(r chi.Router) {
			r.Get("/", serviceHandler.ListServices)
			r.Get("/{id}", serviceHandler.GetService)
		})
	})

	// Start HTTP server
	addr := ":" + httpPort
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		logger.Info("starting admin service", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	logger.Info("server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
