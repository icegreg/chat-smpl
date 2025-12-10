package centrifugo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/icegreg/chat-smpl/pkg/logger"
	"go.uber.org/zap"
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

	logger.Debug("broadcast to centrifugo",
		zap.Int("channels", len(channels)),
	)

	return nil
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

	logger.Debug("published to centrifugo",
		zap.String("channel", channel),
	)

	return nil
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
