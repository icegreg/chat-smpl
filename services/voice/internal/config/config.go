package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Server
	GRPCAddr string
	HTTPPort int

	// Database
	DatabaseURL string

	// RabbitMQ
	RabbitMQURL string

	// Chat service gRPC
	ChatServiceAddr string

	// FreeSWITCH ESL
	FreeSWITCH FreeSWITCHConfig

	// TURN server
	TURN TURNConfig

	// Verto
	Verto VertoConfig

	// JWT
	JWTSecret string

	// Conference cleanup settings
	EmptyConferenceTimeout int // seconds - timeout for empty conferences before auto-end
}

type FreeSWITCHConfig struct {
	ESLHost     string
	ESLPort     int
	ESLPassword string
}

type TURNConfig struct {
	URLs       []string
	Username   string
	Credential string
}

type VertoConfig struct {
	WSUrl          string
	Domain         string
	CredentialsTTL int // seconds
}

func Load() (*Config, error) {
	eslPort, _ := strconv.Atoi(getEnv("FREESWITCH_ESL_PORT", "8021"))
	httpPort, _ := strconv.Atoi(getEnv("HTTP_PORT", "8084"))
	credentialsTTL, _ := strconv.Atoi(getEnv("VERTO_CREDENTIALS_TTL", "3600"))
	emptyConfTimeout, _ := strconv.Atoi(getEnv("EMPTY_CONFERENCE_TIMEOUT", "30"))

	turnURLs := strings.Split(getEnv("TURN_URLS", "turn:localhost:3478"), ",")

	return &Config{
		GRPCAddr: getEnv("GRPC_ADDR", ":50054"),
		HTTPPort: httpPort,

		DatabaseURL:     getEnv("DATABASE_URL", "postgres://chatapp:secret@localhost:5435/chatapp?sslmode=disable"),
		RabbitMQURL:     getEnv("RABBITMQ_URL", "amqp://chatapp:secret@localhost:5672/"),
		ChatServiceAddr: getEnv("CHAT_SERVICE_ADDR", "chat-service:50051"),

		FreeSWITCH: FreeSWITCHConfig{
			ESLHost:     getEnv("FREESWITCH_ESL_HOST", "localhost"),
			ESLPort:     eslPort,
			ESLPassword: getEnv("FREESWITCH_ESL_PASSWORD", "ClueCon"),
		},

		TURN: TURNConfig{
			URLs:       turnURLs,
			Username:   getEnv("TURN_USERNAME", "chatapp"),
			Credential: getEnv("TURN_CREDENTIAL", "chatapp_turn_secret"),
		},

		Verto: VertoConfig{
			WSUrl:          getEnv("VERTO_WS_URL", "ws://localhost:8081"),
			Domain:         getEnv("VERTO_DOMAIN", "chatapp.local"),
			CredentialsTTL: credentialsTTL,
		},

		JWTSecret:              getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		EmptyConferenceTimeout: emptyConfTimeout,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
