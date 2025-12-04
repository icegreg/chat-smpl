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
	"github.com/icegreg/chat-smpl/pkg/postgres"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	pb "github.com/icegreg/chat-smpl/proto/chat"
	"github.com/icegreg/chat-smpl/services/chat/internal/events"
	chatgrpc "github.com/icegreg/chat-smpl/services/chat/internal/grpc"
	"github.com/icegreg/chat-smpl/services/chat/internal/repository"
	"github.com/icegreg/chat-smpl/services/chat/internal/service"
	"go.uber.org/zap"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string
	RabbitMQURL string
}

func loadConfig() Config {
	return Config{
		GRPCPort:    getEnv("GRPC_PORT", "50051"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5432/chatapp?sslmode=disable"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://chatapp:secret@localhost:5672/"),
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

	ctx := context.Background()

	// Connect to database
	pool, err := postgres.NewPool(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer postgres.Close(pool)

	// Connect to RabbitMQ
	var publisher events.Publisher
	rmqConn, err := rabbitmq.NewConnection(rabbitmq.Config{URL: cfg.RabbitMQURL})
	if err != nil {
		logger.Warn("failed to connect to RabbitMQ, using no-op publisher", zap.Error(err))
		publisher = events.NewNoOpPublisher()
	} else {
		defer rmqConn.Close()
		publisher, err = events.NewPublisher(rmqConn)
		if err != nil {
			logger.Warn("failed to create publisher, using no-op publisher", zap.Error(err))
			publisher = events.NewNoOpPublisher()
		}
	}

	// Initialize layers
	chatRepo := repository.NewChatRepository(pool)
	chatService := service.NewChatService(chatRepo, publisher)
	chatServer := chatgrpc.NewChatServer(chatService)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor),
	)

	// Register service
	pb.RegisterChatServiceServer(grpcServer, chatServer)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start server
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	logger.Info("starting chat service", zap.String("port", cfg.GRPCPort))
	if err := grpcServer.Serve(listener); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}

	logger.Info("server stopped")
}

func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	logger.Debug("gRPC request",
		zap.String("method", info.FullMethod),
	)

	resp, err := handler(ctx, req)

	if err != nil {
		logger.Error("gRPC error",
			zap.String("method", info.FullMethod),
			zap.Error(err),
		)
	}

	return resp, err
}
