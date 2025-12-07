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

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/services/api-gateway/internal/centrifugo"
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
	chatHandler := handler.NewChatHandler(chatClient, filesClient, log)
	centrifugoHandler := handler.NewCentrifugoHandler(centrifugoClient, log)
	filesHandler := handler.NewFilesHandler(filesServiceURL, log)

	// Create router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(middleware.CORS)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

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
