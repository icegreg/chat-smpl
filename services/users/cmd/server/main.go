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

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/metrics"
	"github.com/icegreg/chat-smpl/pkg/postgres"
	"github.com/icegreg/chat-smpl/services/users/internal/handler"
	"github.com/icegreg/chat-smpl/services/users/internal/repository"
	"github.com/icegreg/chat-smpl/services/users/internal/service"
	"go.uber.org/zap"
)

type Config struct {
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	AvatarsPath string
}

func loadConfig() Config {
	return Config{
		HTTPPort:    getEnv("HTTP_PORT", "8081"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5432/chatapp?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		AvatarsPath: getEnv("AVATARS_PATH", "./avatars"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	logger.InitDefault()
	defer logger.Sync()

	cfg := loadConfig()

	// Connect to database
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer postgres.Close(pool)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(jwt.DefaultConfig(cfg.JWTSecret))

	// Initialize layers
	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserService(userRepo, jwtManager, cfg.AvatarsPath)
	h := handler.New(userService, jwtManager)

	// Check and regenerate missing avatars on startup
	logger.Info("checking for missing avatar files...")
	regenerated, err := userService.RegenerateMissingAvatars(ctx)
	if err != nil {
		logger.Error("failed to regenerate missing avatars", zap.Error(err))
	} else if regenerated > 0 {
		logger.Info("regenerated missing avatars", zap.Int("count", regenerated))
	} else {
		logger.Info("all avatars present")
	}

	// Setup router
	r := chi.NewRouter()

	// Initialize metrics
	httpMetrics := metrics.NewHTTPMetrics("users_service")

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(metrics.HTTPMiddleware(httpMetrics))

	// CORS
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")

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

	// Prometheus metrics endpoint
	r.Handle("/metrics", metrics.Handler())

	// Register routes
	h.RegisterRoutes(r)

	// Start server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown error", zap.Error(err))
		}
	}()

	logger.Info("starting users service", zap.String("port", cfg.HTTPPort))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}

	logger.Info("server stopped")
}
