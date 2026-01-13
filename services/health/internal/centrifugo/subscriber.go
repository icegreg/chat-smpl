package centrifugo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt/v5"
)

type ReceivedMessage struct {
	MessageID  string
	ReceivedAt time.Time
	Data       json.RawMessage
}

type Subscriber struct {
	wsURL        string
	secret       string
	userID       string
	client       *centrifuge.Client
	subscription *centrifuge.Subscription
	log          *slog.Logger

	mu          sync.RWMutex
	messagesCh  chan ReceivedMessage
	isConnected bool
}

func NewSubscriber(wsURL, secret, userID string, log *slog.Logger) *Subscriber {
	return &Subscriber{
		wsURL:      wsURL,
		secret:     secret,
		userID:     userID,
		log:        log,
		messagesCh: make(chan ReceivedMessage, 100),
	}
}

func (s *Subscriber) generateToken() (string, error) {
	claims := jwt.MapClaims{
		"sub": s.userID,
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *Subscriber) generateSubscriptionToken(channel string) (string, error) {
	claims := jwt.MapClaims{
		"sub":     s.userID,
		"channel": channel,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *Subscriber) Connect(ctx context.Context) error {
	token, err := s.generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	s.client = centrifuge.NewJsonClient(s.wsURL, centrifuge.Config{
		Token: token,
		GetToken: func(event centrifuge.ConnectionTokenEvent) (string, error) {
			return s.generateToken()
		},
	})

	s.client.OnConnected(func(e centrifuge.ConnectedEvent) {
		s.mu.Lock()
		s.isConnected = true
		s.mu.Unlock()
		s.log.Info("connected to Centrifugo", "client_id", e.ClientID)
	})

	s.client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		s.mu.Lock()
		s.isConnected = false
		s.mu.Unlock()
		s.log.Warn("disconnected from Centrifugo", "code", e.Code, "reason", e.Reason)
	})

	s.client.OnError(func(e centrifuge.ErrorEvent) {
		s.log.Error("Centrifugo client error", "error", e.Error)
	})

	err = s.client.Connect()
	if err != nil {
		return fmt.Errorf("connect to Centrifugo: %w", err)
	}

	// Subscribe to user channel
	channel := fmt.Sprintf("user:%s", s.userID)
	subToken, err := s.generateSubscriptionToken(channel)
	if err != nil {
		return fmt.Errorf("generate subscription token: %w", err)
	}

	s.subscription, err = s.client.NewSubscription(channel, centrifuge.SubscriptionConfig{
		Token: subToken,
		GetToken: func(event centrifuge.SubscriptionTokenEvent) (string, error) {
			return s.generateSubscriptionToken(event.Channel)
		},
	})
	if err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	s.subscription.OnPublication(func(e centrifuge.PublicationEvent) {
		s.log.Debug("received publication", "channel", channel, "data_len", len(e.Data))

		// Parse the event to extract message ID
		var event struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(e.Data, &event); err != nil {
			s.log.Error("failed to parse event", "error", err)
			return
		}

		if event.Type == "message.created" {
			var msgData struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(event.Data, &msgData); err != nil {
				s.log.Error("failed to parse message data", "error", err)
				return
			}

			select {
			case s.messagesCh <- ReceivedMessage{
				MessageID:  msgData.ID,
				ReceivedAt: time.Now(),
				Data:       e.Data,
			}:
			default:
				s.log.Warn("message channel full, dropping message")
			}
		}
	})

	s.subscription.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		s.log.Info("subscribed to channel", "channel", channel)
	})

	s.subscription.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		s.log.Error("subscription error", "error", e.Error)
	})

	err = s.subscription.Subscribe()
	if err != nil {
		return fmt.Errorf("subscribe to channel: %w", err)
	}

	return nil
}

func (s *Subscriber) WaitForMessage(ctx context.Context, messageID string, timeout time.Duration) (*ReceivedMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for message %s", messageID)
		case msg := <-s.messagesCh:
			if msg.MessageID == messageID {
				return &msg, nil
			}
			// Not our message, continue waiting
			s.log.Debug("received different message", "expected", messageID, "got", msg.MessageID)
		}
	}
}

func (s *Subscriber) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isConnected
}

func (s *Subscriber) Close() error {
	if s.subscription != nil {
		s.subscription.Unsubscribe()
	}
	if s.client != nil {
		s.client.Close()
	}
	return nil
}
