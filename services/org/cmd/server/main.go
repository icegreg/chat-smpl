package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/migrate"
	"github.com/icegreg/chat-smpl/pkg/postgres"
	pb "github.com/icegreg/chat-smpl/proto/org"
	migrations "github.com/icegreg/chat-smpl/services/org/migrations"
	orggrpc "github.com/icegreg/chat-smpl/services/org/internal/grpc"
	"github.com/icegreg/chat-smpl/services/org/internal/repository"
	"github.com/icegreg/chat-smpl/services/org/internal/service"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string
}

func loadConfig() Config {
	return Config{
		GRPCPort:    getEnv("GRPC_PORT", "50055"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5432/chatapp?sslmode=disable"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log := logger.New(logger.Config{
		Level:       "info",
		Development: true,
	})

	cfg := loadConfig()
	ctx := context.Background()

	// Connect to database
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		log.Fatal("failed to connect to database", "error", err)
	}
	defer pool.Close()

	log.Info("connected to database")

	// Run database migrations
	if err := migrate.RunWithDSN(cfg.DatabaseURL, migrate.Config{
		ServiceName:    "org",
		MigrationsFS:   migrations.FS,
		MigrationsPath: ".",
	}); err != nil {
		log.Fatal("failed to run migrations", "error", err)
	}

	// Initialize layers
	orgRepo := repository.NewOrgRepository(pool)
	orgService := service.NewOrgService(orgRepo)
	orgServer := orggrpc.NewOrgServer(orgService)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterOrgServiceServer(grpcServer, orgServer)
	reflection.Register(grpcServer)

	// Start server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatal("failed to listen", "error", err)
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Info("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Info("starting org service", "port", cfg.GRPCPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("failed to serve", "error", err)
	}
}
