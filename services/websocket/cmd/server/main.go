package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/rabbitmq"
	"github.com/icegreg/chat-smpl/services/websocket/internal/centrifugo"
	"github.com/icegreg/chat-smpl/services/websocket/internal/consumer"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger.InitDefault()
	defer logger.Sync()

	logger.Info("starting websocket-service")

	// Get configuration from environment
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://chatapp:secret@localhost:5672/")
	centrifugoAPIURL := getEnv("CENTRIFUGO_API_URL", "http://localhost:8000/api")
	centrifugoAPIKey := getEnv("CENTRIFUGO_API_KEY", "centrifugo-api-key")
	centrifugoSecret := getEnv("CENTRIFUGO_SECRET", "your-centrifugo-secret-key-change-in-production")

	// Connect to RabbitMQ
	rmqConn, err := rabbitmq.NewConnection(rabbitmq.Config{
		URL: rabbitURL,
	})
	if err != nil {
		logger.Fatal("failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rmqConn.Close()

	// Create Centrifugo client
	centrifugoClient := centrifugo.NewClient(centrifugo.Config{
		APIURL:     centrifugoAPIURL,
		APIKey:     centrifugoAPIKey,
		HMACSecret: centrifugoSecret,
	})

	// Create chat consumer
	chatConsumer := consumer.New(rmqConn, centrifugoClient)

	// Setup chat queue bindings
	if err := chatConsumer.Setup(); err != nil {
		logger.Fatal("failed to setup chat consumer", zap.Error(err))
	}

	// Create voice consumer
	voiceConsumer := consumer.NewVoiceConsumer(rmqConn, centrifugoClient)

	// Setup voice queue bindings
	if err := voiceConsumer.Setup(); err != nil {
		logger.Fatal("failed to setup voice consumer", zap.Error(err))
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		cancel()
	}()

	// Start chat consumer in goroutine
	go func() {
		logger.Info("starting chat consumer...")
		if err := chatConsumer.Start(ctx); err != nil && err != context.Canceled {
			logger.Error("chat consumer error", zap.Error(err))
		}
	}()

	// Start voice consumer
	logger.Info("websocket-service started, waiting for events...")
	if err := voiceConsumer.Start(ctx); err != nil && err != context.Canceled {
		logger.Error("voice consumer error", zap.Error(err))
	}

	logger.Info("websocket-service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
