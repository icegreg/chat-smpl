package centrifugo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"go.uber.org/zap"
)

// Retry configuration
const (
	maxRetries     = 3
	baseRetryDelay = 100 * time.Millisecond
	maxRetryDelay  = 2 * time.Second
)

type Config struct {
	APIURL    string
	APIKey    string
	HMACSecret string
}

type Client struct {
	httpClient *http.Client
	apiURL     string
	apiKey     string
	hmacSecret string
}

func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiURL:     cfg.APIURL,
		apiKey:     cfg.APIKey,
		hmacSecret: cfg.HMACSecret,
	}
}

type publishRequest struct {
	Channel string      `json:"channel"`
	Data    interface{} `json:"data"`
}

type broadcastRequest struct {
	Channels []string    `json:"channels"`
	Data     interface{} `json:"data"`
}

type apiRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type apiResponse struct {
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PublishToUser publishes an event to a user's personal channel
func (c *Client) PublishToUser(ctx context.Context, userID string, event interface{}) error {
	channel := fmt.Sprintf("user:%s", userID)
	return c.publish(ctx, channel, event)
}

// PublishToChannel publishes an event to a specific channel
func (c *Client) PublishToChannel(ctx context.Context, channel string, event interface{}) error {
	return c.publish(ctx, channel, event)
}

// BroadcastToUsers broadcasts an event to multiple users' personal channels in a single request
func (c *Client) BroadcastToUsers(ctx context.Context, userIDs []string, event interface{}) error {
	if len(userIDs) == 0 {
		return nil
	}

	// Convert user IDs to channels
	channels := make([]string, len(userIDs))
	for i, userID := range userIDs {
		channels[i] = fmt.Sprintf("user:%s", userID)
	}

	return c.broadcast(ctx, channels, event)
}

// Broadcast sends an event to multiple channels in a single request
func (c *Client) Broadcast(ctx context.Context, channels []string, event interface{}) error {
	return c.broadcast(ctx, channels, event)
}

func (c *Client) broadcast(ctx context.Context, channels []string, data interface{}) error {
	req := apiRequest{
		Method: "broadcast",
		Params: broadcastRequest{
			Channels: channels,
			Data:     data,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoff(attempt)
			logger.Warn("retrying centrifugo broadcast",
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
				zap.Error(lastErr),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doRequest(ctx, body)
		if err == nil {
			if attempt > 0 {
				logger.Info("centrifugo broadcast succeeded after retry",
					zap.Int("attempts", attempt+1),
					zap.Int("channels", len(channels)),
				)
			} else {
				logger.Debug("broadcast to centrifugo",
					zap.Int("channels", len(channels)),
				)
			}
			return nil
		}

		lastErr = err
		if !c.isRetryableError(err) {
			return err
		}
	}

	logger.Error("centrifugo broadcast failed after retries",
		zap.Int("maxRetries", maxRetries),
		zap.Int("channels", len(channels)),
		zap.Error(lastErr),
	)
	return fmt.Errorf("broadcast failed after %d retries: %w", maxRetries, lastErr)
}

func (c *Client) publish(ctx context.Context, channel string, data interface{}) error {
	req := apiRequest{
		Method: "publish",
		Params: publishRequest{
			Channel: channel,
			Data:    data,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoff(attempt)
			logger.Warn("retrying centrifugo publish",
				zap.Int("attempt", attempt),
				zap.String("channel", channel),
				zap.Duration("delay", delay),
				zap.Error(lastErr),
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := c.doRequest(ctx, body)
		if err == nil {
			if attempt > 0 {
				logger.Info("centrifugo publish succeeded after retry",
					zap.Int("attempts", attempt+1),
					zap.String("channel", channel),
				)
			} else {
				logger.Debug("published to centrifugo",
					zap.String("channel", channel),
				)
			}
			return nil
		}

		lastErr = err
		if !c.isRetryableError(err) {
			return err
		}
	}

	logger.Error("centrifugo publish failed after retries",
		zap.Int("maxRetries", maxRetries),
		zap.String("channel", channel),
		zap.Error(lastErr),
	)
	return fmt.Errorf("publish failed after %d retries: %w", maxRetries, lastErr)
}

// doRequest executes a single HTTP request to Centrifugo API
func (c *Client) doRequest(ctx context.Context, body []byte) error {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "apikey "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: status code %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Error != nil {
		return fmt.Errorf("centrifugo error: %s (code: %d)", apiResp.Error.Message, apiResp.Error.Code)
	}

	return nil
}

// isRetryableError determines if an error should trigger a retry
func (c *Client) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are retryable
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "server error: status code 5") {
		return true
	}

	// Check for net.Error (timeout, temporary)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	return false
}

// calculateBackoff calculates delay with exponential backoff and jitter
func (c *Client) calculateBackoff(attempt int) time.Duration {
	delay := baseRetryDelay * time.Duration(1<<uint(attempt-1)) // 100ms, 200ms, 400ms...
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}

	// Add jitter (±25%)
	jitter := time.Duration(rand.Int63n(int64(delay / 2)))
	if rand.Intn(2) == 0 {
		delay += jitter
	} else {
		delay -= jitter / 2
	}

	return delay
}

// GenerateUserToken generates a connection token for a user
func (c *Client) GenerateUserToken(userID string, expireAt int64) string {
	claims := map[string]interface{}{
		"sub": userID,
	}
	if expireAt > 0 {
		claims["exp"] = expireAt
	}

	return c.generateToken(claims)
}

// GenerateSubscriptionToken generates a subscription token for a channel
func (c *Client) GenerateSubscriptionToken(userID, channel string, expireAt int64) string {
	claims := map[string]interface{}{
		"sub":     userID,
		"channel": channel,
	}
	if expireAt > 0 {
		claims["exp"] = expireAt
	}

	return c.generateToken(claims)
}

func (c *Client) generateToken(claims map[string]interface{}) string {
	header := map[string]string{
		"typ": "JWT",
		"alg": "HS256",
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	signatureInput := headerB64 + "." + claimsB64

	h := hmac.New(sha256.New, []byte(c.hmacSecret))
	h.Write([]byte(signatureInput))
	signature := h.Sum(nil)

	signatureB64 := base64URLEncode(signature)

	return signatureInput + "." + signatureB64
}

func base64URLEncode(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	// Convert to URL-safe base64
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.TrimRight(encoded, "=")
	return encoded
}
