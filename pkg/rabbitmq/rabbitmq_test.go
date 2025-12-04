package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		URL: "amqp://guest:guest@localhost:5672/",
	}

	assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.URL)
}

func TestExchange(t *testing.T) {
	ex := Exchange{
		Name:       "test.exchange",
		Kind:       "topic",
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
		Args:       nil,
	}

	assert.Equal(t, "test.exchange", ex.Name)
	assert.Equal(t, "topic", ex.Kind)
	assert.True(t, ex.Durable)
}

func TestQueue(t *testing.T) {
	q := Queue{
		Name:       "test.queue",
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}

	assert.Equal(t, "test.queue", q.Name)
	assert.True(t, q.Durable)
}

// Integration tests require a running RabbitMQ instance
// and are skipped by default. Run with: go test -tags=integration ./pkg/rabbitmq/...
