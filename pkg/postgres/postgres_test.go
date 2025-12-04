package postgres

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	url := "postgres://user:pass@localhost:5432/db"
	cfg := DefaultConfig(url)

	assert.Equal(t, url, cfg.URL)
	assert.Equal(t, int32(25), cfg.MaxConns)
	assert.Equal(t, int32(5), cfg.MinConns)
	assert.Equal(t, time.Hour, cfg.MaxConnLifetime)
	assert.Equal(t, 30*time.Minute, cfg.MaxConnIdleTime)
}

// Integration tests require a running PostgreSQL instance
// and are skipped by default. Run with: go test -tags=integration ./pkg/postgres/...
