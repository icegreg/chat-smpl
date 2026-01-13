package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort              string
	CheckInterval         time.Duration
	MessageTimeout        time.Duration
	ChatServiceAddr       string
	VoiceServiceAddr      string
	CentrifugoWSURL       string
	CentrifugoSecret      string
	SystemUserID          string
	SystemChatID          string
	SystemUser2ID         string // Additional test user for voice
	SystemUser3ID         string // Additional test user for voice
	DegradedThresholdMs   int64
	DownThresholdMs       int64
	ConsecutiveFailures   int
	VoiceCheckEnabled     bool
}

func Load() *Config {
	return &Config{
		HTTPPort:              getEnv("HTTP_PORT", ":8085"),
		CheckInterval:         getDurationEnv("CHECK_INTERVAL", 5*time.Second),
		MessageTimeout:        getDurationEnv("MESSAGE_TIMEOUT", 10*time.Second),
		ChatServiceAddr:       getEnv("CHAT_SERVICE_ADDR", "chat-service:50051"),
		VoiceServiceAddr:      getEnv("VOICE_SERVICE_ADDR", "voice-service:50054"),
		CentrifugoWSURL:       getEnv("CENTRIFUGO_WS_URL", "ws://centrifugo:8000/connection/websocket"),
		CentrifugoSecret:      getEnv("CENTRIFUGO_SECRET", "your-centrifugo-secret-key-change-in-production"),
		SystemUserID:          getEnv("SYSTEM_USER_ID", "00000000-0000-0000-0000-000000000001"),
		SystemChatID:          getEnv("SYSTEM_CHAT_ID", "00000000-0000-0000-0000-000000000002"),
		SystemUser2ID:         getEnv("SYSTEM_USER2_ID", "00000000-0000-0000-0000-000000000003"),
		SystemUser3ID:         getEnv("SYSTEM_USER3_ID", "00000000-0000-0000-0000-000000000004"),
		DegradedThresholdMs:   getIntEnv("DEGRADED_THRESHOLD_MS", 2000),
		DownThresholdMs:       getIntEnv("DOWN_THRESHOLD_MS", 5000),
		ConsecutiveFailures:   int(getIntEnv("CONSECUTIVE_FAILURES_FOR_DOWN", 3)),
		VoiceCheckEnabled:     getBoolEnv("VOICE_CHECK_ENABLED", true),
	}
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}
