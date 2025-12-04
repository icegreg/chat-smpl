package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		development bool
	}{
		{"development debug", "debug", true},
		{"development info", "info", true},
		{"production info", "info", false},
		{"production warn", "warn", false},
		{"invalid level defaults to info", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.level, tt.development)
			require.NoError(t, err)

			logger := Get()
			assert.NotNil(t, logger)
		})
	}
}

func TestInitDefault(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("LOG_LEVEL", "debug")

	InitDefault()
	logger := Get()
	assert.NotNil(t, logger)
}

func TestLogFunctions(t *testing.T) {
	err := Init("debug", true)
	require.NoError(t, err)

	// These should not panic
	assert.NotPanics(t, func() {
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")
	})
}

func TestWith(t *testing.T) {
	err := Init("debug", true)
	require.NoError(t, err)

	logger := With()
	assert.NotNil(t, logger)
}
