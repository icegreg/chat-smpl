package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"go.uber.org/zap"
)

type Config struct {
	URL string
}

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.RWMutex
	url     string
	done    chan struct{}
}

func NewConnection(cfg Config) (*Connection, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	c := &Connection{
		conn:    conn,
		channel: ch,
		url:     cfg.URL,
		done:    make(chan struct{}),
	}

	go c.handleReconnect()

	logger.Info("connected to RabbitMQ")
	return c, nil
}

func (c *Connection) handleReconnect() {
	for {
		select {
		case <-c.done:
			return
		case err := <-c.conn.NotifyClose(make(chan *amqp.Error)):
			if err != nil {
				logger.Error("RabbitMQ connection lost", zap.Error(err))
				c.reconnect()
			}
		}
	}
}

func (c *Connection) reconnect() {
	for {
		select {
		case <-c.done:
			return
		default:
			logger.Info("attempting to reconnect to RabbitMQ...")

			conn, err := amqp.Dial(c.url)
			if err != nil {
				logger.Error("failed to reconnect to RabbitMQ", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}

			ch, err := conn.Channel()
			if err != nil {
				conn.Close()
				logger.Error("failed to open channel", zap.Error(err))
				time.Sleep(5 * time.Second)
				continue
			}

			c.mu.Lock()
			c.conn = conn
			c.channel = ch
			c.mu.Unlock()

			logger.Info("reconnected to RabbitMQ")
			return
		}
	}
}

func (c *Connection) Channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channel
}

func (c *Connection) Close() error {
	close(c.done)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type Exchange struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

func (c *Connection) DeclareExchange(ex Exchange) error {
	return c.Channel().ExchangeDeclare(
		ex.Name,
		ex.Kind,
		ex.Durable,
		ex.AutoDelete,
		ex.Internal,
		ex.NoWait,
		ex.Args,
	)
}

type Queue struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

func (c *Connection) DeclareQueue(q Queue) (amqp.Queue, error) {
	return c.Channel().QueueDeclare(
		q.Name,
		q.Durable,
		q.AutoDelete,
		q.Exclusive,
		q.NoWait,
		q.Args,
	)
}

func (c *Connection) BindQueue(queueName, routingKey, exchangeName string) error {
	return c.Channel().QueueBind(
		queueName,
		routingKey,
		exchangeName,
		false,
		nil,
	)
}

type Publisher struct {
	conn     *Connection
	exchange string
}

func NewPublisher(conn *Connection, exchange string) *Publisher {
	return &Publisher{
		conn:     conn,
		exchange: exchange,
	}
}

type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, event interface{}) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return p.conn.Channel().PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)
}

type Consumer struct {
	conn       *Connection
	queue      string
	consumer   string
	prefetch   int
	numWorkers int
}

type ConsumerOption func(*Consumer)

// WithPrefetch sets the prefetch count (QoS) for the consumer
func WithPrefetch(prefetch int) ConsumerOption {
	return func(c *Consumer) {
		c.prefetch = prefetch
	}
}

// WithWorkers sets the number of parallel workers
func WithWorkers(numWorkers int) ConsumerOption {
	return func(c *Consumer) {
		c.numWorkers = numWorkers
	}
}

func NewConsumer(conn *Connection, queue, consumer string, opts ...ConsumerOption) *Consumer {
	c := &Consumer{
		conn:       conn,
		queue:      queue,
		consumer:   consumer,
		prefetch:   1,  // default: 1 message at a time
		numWorkers: 1,  // default: single worker
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type MessageHandler func(ctx context.Context, msg amqp.Delivery) error

func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	// Set QoS (prefetch)
	if err := c.conn.Channel().Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	logger.Info("consumer QoS configured",
		zap.Int("prefetch", c.prefetch),
		zap.Int("workers", c.numWorkers),
	)

	msgs, err := c.conn.Channel().Consume(
		c.queue,
		c.consumer,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	// Single worker mode (original behavior)
	if c.numWorkers <= 1 {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-msgs:
				if !ok {
					return fmt.Errorf("channel closed")
				}
				c.processMessage(ctx, msg, handler)
			}
		}
	}

	// Worker pool mode
	return c.consumeWithWorkerPool(ctx, msgs, handler)
}

func (c *Consumer) consumeWithWorkerPool(ctx context.Context, msgs <-chan amqp.Delivery, handler MessageHandler) error {
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < c.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logger.Debug("worker started", zap.Int("worker_id", workerID))

			for {
				select {
				case <-ctx.Done():
					logger.Debug("worker stopping", zap.Int("worker_id", workerID))
					return
				case msg, ok := <-msgs:
					if !ok {
						logger.Debug("worker channel closed", zap.Int("worker_id", workerID))
						return
					}
					c.processMessage(ctx, msg, handler)
				}
			}
		}(i)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Wait for all workers to finish
	wg.Wait()
	return ctx.Err()
}

func (c *Consumer) processMessage(ctx context.Context, msg amqp.Delivery, handler MessageHandler) {
	if err := handler(ctx, msg); err != nil {
		logger.Error("failed to handle message",
			zap.Error(err),
			zap.String("routing_key", msg.RoutingKey),
		)
		msg.Nack(false, true)
	} else {
		msg.Ack(false)
	}
}
