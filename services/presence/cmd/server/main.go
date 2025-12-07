package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	pb "github.com/icegreg/chat-smpl/proto/presence"
	"github.com/icegreg/chat-smpl/services/presence/internal/handler"
	"github.com/icegreg/chat-smpl/services/presence/internal/repository"
	"github.com/icegreg/chat-smpl/services/presence/internal/service"
)

func main() {
	// Initialize logger
	log := logger.New(logger.Config{
		Level:       getEnv("LOG_LEVEL", "info"),
		Development: getEnv("ENV", "development") == "development",
	})
	defer log.Sync()

	log.Info("starting presence-service")

	// Connect to Redis
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("failed to connect to Redis", "error", err, "addr", redisAddr)
	}
	defer redisClient.Close()
	log.Info("connected to Redis", "addr", redisAddr)

	// Connect to RabbitMQ (optional, for publishing events)
	var rmqConn *rabbitmq.Connection
	rabbitURL := getEnv("RABBITMQ_URL", "")
	if rabbitURL != "" {
		var err error
		rmqConn, err = rabbitmq.NewConnection(rabbitmq.Config{
			URL: rabbitURL,
		})
		if err != nil {
			log.Warn("failed to connect to RabbitMQ, presence events won't be published", "error", err)
		} else {
			defer rmqConn.Close()
			log.Info("connected to RabbitMQ")
		}
	}

	// Initialize repository and service
	repo := repository.NewRepository(redisClient)

	var svc *service.Service
	if rmqConn != nil {
		svc = service.NewService(repo, rmqConn.Channel())
		if err := svc.SetupExchange(); err != nil {
			log.Warn("failed to setup RabbitMQ exchange", "error", err)
		}
	} else {
		svc = service.NewService(repo, nil)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler := handler.NewGRPCHandler(svc)
	pb.RegisterPresenceServiceServer(grpcServer, grpcHandler)

	// Enable reflection for development
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcAddr := getEnv("GRPC_ADDR", ":50052")
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("failed to listen", "error", err, "addr", grpcAddr)
	}

	// Graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Info("shutting down...")
		grpcServer.GracefulStop()
		close(done)
	}()

	log.Info("presence-service started", "grpc_addr", grpcAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("failed to serve", "error", err)
	}

	<-done
	log.Info("presence-service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
