// @title Chat-SMPL API
// @version 1.0.0
// @description Микросервисное чат-приложение с JWT авторизацией, gRPC коммуникацией и real-time обновлениями
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@chat-smpl.local

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8888
// @BasePath /api

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description JWT Bearer token. Format: "Bearer {token}"

// @tag.name auth
// @tag.description Аутентификация и управление пользователями
// @tag.name chats
// @tag.description Операции с чатами
// @tag.name messages
// @tag.description Операции с сообщениями
// @tag.name threads
// @tag.description Операции с потоками (threads)
// @tag.name files
// @tag.description Загрузка и скачивание файлов
// @tag.name presence
// @tag.description Статус присутствия пользователей
// @tag.name centrifugo
// @tag.description WebSocket токены для real-time

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/metrics"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/centrifugo"
	_ "github.com/icegreg/chat-smpl/services/api-gateway/docs" // swagger docs
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/files"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/grpc"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/handler"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/middleware"
)

func main() {
	// Initialize logger
	log := logger.New(logger.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Development: getEnv("ENV", "development") == "development",
	})
	defer log.Sync()

	// Initialize JWT manager
	jwtManager := jwt.NewManager(jwt.Config{
		Secret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "chat-smpl",
	})

	// Initialize gRPC chat client
	chatServiceAddr := getEnv("CHAT_SERVICE_ADDR", "localhost:50051")
	chatClient, err := grpc.NewChatClient(chatServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to chat service", "error", err)
	}
	defer chatClient.Close()

	log.Info("connected to chat service", "addr", chatServiceAddr)

	// Initialize gRPC presence client
	presenceServiceAddr := getEnv("PRESENCE_SERVICE_ADDR", "localhost:50052")
	presenceClient, err := grpc.NewPresenceClient(presenceServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to presence service", "error", err)
	}
	defer presenceClient.Close()

	log.Info("connected to presence service", "addr", presenceServiceAddr)

	// Initialize gRPC voice client
	voiceServiceAddr := getEnv("VOICE_SERVICE_ADDR", "localhost:50054")
	voiceClient, err := grpc.NewVoiceClient(voiceServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to voice service", "error", err)
	}
	defer voiceClient.Close()

	log.Info("connected to voice service", "addr", voiceServiceAddr)

	// Initialize gRPC org client
	orgServiceAddr := getEnv("ORG_SERVICE_ADDR", "localhost:50055")
	orgClient, err := grpc.NewOrgClient(orgServiceAddr)
	if err != nil {
		log.Fatal("failed to connect to org service", "error", err)
	}
	defer orgClient.Close()

	log.Info("connected to org service", "addr", orgServiceAddr)

	// Initialize Centrifugo client
	centrifugoAPIURL := getEnv("CENTRIFUGO_API_URL", "http://localhost:8000/api")
	centrifugoAPIKey := getEnv("CENTRIFUGO_API_KEY", "your-api-key")
	centrifugoSecret := getEnv("CENTRIFUGO_SECRET", "your-secret-key")
	centrifugoClient := centrifugo.NewClient(centrifugoAPIURL, centrifugoAPIKey, centrifugoSecret)

	log.Info("centrifugo client initialized", "api_url", centrifugoAPIURL)

	// Initialize files service client
	filesServiceURL := getEnv("FILES_SERVICE_URL", "http://localhost:8082")
	filesClient := files.NewClient(filesServiceURL)

	log.Info("files client initialized", "url", filesServiceURL)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Initialize handlers
	usersServiceURL := getEnv("USERS_SERVICE_URL", "http://localhost:8081")
	authHandler := handler.NewAuthHandler(usersServiceURL, centrifugoClient, log)
	chatHandler := handler.NewChatHandler(chatClient, filesClient, orgClient, log)
	centrifugoHandler := handler.NewCentrifugoHandler(centrifugoClient, log)
	filesHandler := handler.NewFilesHandler(filesServiceURL, log)
	presenceHandler := handler.NewPresenceHandler(presenceClient, log)
	voiceHandler := handler.NewVoiceHandler(voiceClient, log)
	eventsHandler := handler.NewEventsHandler()

	// Create router
	r := chi.NewRouter()

	// Initialize metrics
	httpMetrics := metrics.NewHTTPMetrics("api_gateway")

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(middleware.CORS)
	r.Use(metrics.HTTPMiddleware(httpMetrics))

	// Prometheus metrics endpoint
	r.Handle("/metrics", metrics.Handler())

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (public)
		r.Route("/auth", func(r chi.Router) {
			// Public routes
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.RefreshToken)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.Authenticate)
				r.Post("/logout", authHandler.Logout)
				r.Get("/me", authHandler.GetCurrentUser)
				r.Put("/me", authHandler.UpdateCurrentUser)
				r.Put("/me/password", authHandler.ChangePassword)
			})
		})

		// Chat routes (protected)
		r.Route("/chats", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Mount("/", chatHandler.Routes())
		})

		// Centrifugo routes (protected)
		r.Route("/centrifugo", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Mount("/", centrifugoHandler.Routes())
		})

		// Files routes (protected)
		r.Route("/files", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Mount("/", filesHandler.Routes())
		})

		// Presence routes (protected)
		r.Route("/presence", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Mount("/", presenceHandler.Routes())
		})

		// Voice routes (protected)
		r.Route("/voice", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Mount("/", voiceHandler.Routes())
		})

		// Events documentation (public)
		r.Route("/events", func(r chi.Router) {
			r.Mount("/", eventsHandler.Routes())
		})
	})

	// Serve static files for Vue SPA
	staticDir := getEnv("STATIC_DIR", "./web/dist")
	fileServer := http.FileServer(http.Dir(staticDir))

	// SPA handler - serve index.html for all unmatched routes
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Check if the file exists
		path := staticDir + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// Serve index.html for SPA routing
			http.ServeFile(w, r, staticDir+"/index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// Start server
	addr := getEnv("HTTP_ADDR", ":8080")
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Error("could not gracefully shutdown server", "error", err)
		}
		close(done)
	}()

	log.Info("starting api-gateway", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("failed to start server", "error", err)
	}

	<-done
	log.Info("server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
