package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/icegreg/chat-smpl/pkg/logger"
	pb "github.com/icegreg/chat-smpl/proto/files"
	filesgrpc "github.com/icegreg/chat-smpl/services/files/internal/grpc"
	"github.com/icegreg/chat-smpl/services/files/internal/handler"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
	"github.com/icegreg/chat-smpl/services/files/internal/service"
	"github.com/icegreg/chat-smpl/services/files/internal/storage"
)

func main() {
	// Initialize logger
	log := logger.New(logger.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Development: getEnv("ENV", "development") == "development",
	})
	defer log.Sync()

	// Database connection
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/chat_app?sslmode=disable")

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("failed to parse database URL", "error", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatal("failed to create connection pool", "error", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping database", "error", err)
	}

	log.Info("connected to database")

	// Initialize storage
	storagePath := getEnv("STORAGE_PATH", "./uploads")
	localStorage, err := storage.NewLocalStorage(storagePath)
	if err != nil {
		log.Fatal("failed to initialize storage", "error", err)
	}

	log.Info("storage initialized", "path", storagePath)

	// Initialize repository
	repo := repository.NewFileRepository(pool)

	// Initialize service
	baseURL := getEnv("BASE_URL", "http://localhost:8082")
	fileService := service.NewFileService(repo, localStorage, baseURL)

	// Initialize handler
	maxFileSize := int64(100 * 1024 * 1024) // 100MB default
	if envSize := getEnv("MAX_FILE_SIZE_MB", ""); envSize != "" {
		var sizeMB int64
		if _, err := parseEnvInt(envSize, &sizeMB); err == nil {
			maxFileSize = sizeMB * 1024 * 1024
		}
	}

	h := handler.NewHandler(fileService, log, maxFileSize)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-User-ID", "X-Share-Password"},
		ExposedHeaders:   []string{"Content-Disposition", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Mount file routes
	r.Mount("/files", h.Routes())

	// Public avatars endpoint (no auth required)
	r.Get("/avatars/{userId}", h.ServeAvatar)

	// Start HTTP server
	addr := getEnv("HTTP_ADDR", ":8082")
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start gRPC server
	grpcAddr := getEnv("GRPC_ADDR", ":50052")
	grpcServer := grpc.NewServer()
	filesServer := filesgrpc.NewFilesServer(fileService)
	pb.RegisterFilesServiceServer(grpcServer, filesServer)

	// Start gRPC listener
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatal("failed to listen for gRPC", "error", err)
		}
		log.Info("starting gRPC server", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	// Graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("server is shutting down...")

		// Shutdown gRPC server
		grpcServer.GracefulStop()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Error("could not gracefully shutdown server", "error", err)
		}
		close(done)
	}()

	log.Info("starting files service", "addr", addr)
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

func parseEnvInt(value string, target *int64) (bool, error) {
	var n int64
	for _, c := range value {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + int64(c-'0')
	}
	*target = n
	return true, nil
}
