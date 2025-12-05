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
)

type Client struct {
	apiURL    string
	apiKey    string
	secretKey string
	client    *http.Client
}

func NewClient(apiURL, apiKey, secretKey string) *Client {
	return &Client{
		apiURL:    apiURL,
		apiKey:    apiKey,
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GenerateConnectionToken generates a JWT token for Centrifugo connection
func (c *Client) GenerateConnectionToken(userID string, exp int64) string {
	header := base64URLEncode([]byte(`{"typ":"JWT","alg":"HS256"}`))

	payload := map[string]interface{}{
		"sub": userID,
	}
	if exp > 0 {
		payload["exp"] = exp
	}

	payloadBytes, _ := json.Marshal(payload)
	payloadEncoded := base64URLEncode(payloadBytes)

	signatureInput := header + "." + payloadEncoded
	signature := c.sign([]byte(signatureInput))

	return signatureInput + "." + base64URLEncode(signature)
}

// GenerateSubscriptionToken generates a JWT token for channel subscription
func (c *Client) GenerateSubscriptionToken(userID, channel string, exp int64) string {
	header := base64URLEncode([]byte(`{"typ":"JWT","alg":"HS256"}`))

	payload := map[string]interface{}{
		"sub":     userID,
		"channel": channel,
	}
	if exp > 0 {
		payload["exp"] = exp
	}

	payloadBytes, _ := json.Marshal(payload)
	payloadEncoded := base64URLEncode(payloadBytes)

	signatureInput := header + "." + payloadEncoded
	signature := c.sign([]byte(signatureInput))

	return signatureInput + "." + base64URLEncode(signature)
}

func (c *Client) sign(data []byte) []byte {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write(data)
	return h.Sum(nil)
}

// Publish sends a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, data interface{}) error {
	return c.sendCommand(ctx, "publish", map[string]interface{}{
		"channel": channel,
		"data":    data,
	})
}

// Broadcast sends a message to multiple channels
func (c *Client) Broadcast(ctx context.Context, channels []string, data interface{}) error {
	return c.sendCommand(ctx, "broadcast", map[string]interface{}{
		"channels": channels,
		"data":     data,
	})
}

// Unsubscribe removes a user from a channel
func (c *Client) Unsubscribe(ctx context.Context, channel, userID string) error {
	return c.sendCommand(ctx, "unsubscribe", map[string]interface{}{
		"channel": channel,
		"user":    userID,
	})
}

// Disconnect disconnects a user from all channels
func (c *Client) Disconnect(ctx context.Context, userID string) error {
	return c.sendCommand(ctx, "disconnect", map[string]interface{}{
		"user": userID,
	})
}

// Presence returns presence information for a channel
func (c *Client) Presence(ctx context.Context, channel string) (map[string]interface{}, error) {
	resp, err := c.sendCommandWithResponse(ctx, "presence", map[string]interface{}{
		"channel": channel,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) sendCommand(ctx context.Context, method string, params map[string]interface{}) error {
	_, err := c.sendCommandWithResponse(ctx, method, params)
	return err
}

func (c *Client) sendCommandWithResponse(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"method": method,
		"params": params,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "apikey "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("centrifugo returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if errMsg, ok := result["error"].(map[string]interface{}); ok {
		return nil, fmt.Errorf("centrifugo error: %v", errMsg["message"])
	}

	if resultData, ok := result["result"].(map[string]interface{}); ok {
		return resultData, nil
	}

	return result, nil
}

func base64URLEncode(data []byte) string {
	// JWT requires Base64URL encoding (no padding, URL-safe characters)
	encoded := base64.StdEncoding.EncodeToString(data)
	// Convert to URL-safe base64: replace + with -, / with _, and remove padding
	encoded = strings.ReplaceAll(encoded, "+", "-")
	encoded = strings.ReplaceAll(encoded, "/", "_")
	encoded = strings.TrimRight(encoded, "=")
	return encoded
}

// Event types for real-time updates
type EventType string

const (
	EventNewMessage     EventType = "message.new"
	EventUpdateMessage  EventType = "message.update"
	EventDeleteMessage  EventType = "message.delete"
	EventNewChat        EventType = "chat.new"
	EventUpdateChat     EventType = "chat.update"
	EventDeleteChat     EventType = "chat.delete"
	EventTyping         EventType = "typing"
	EventUserOnline     EventType = "user.online"
	EventUserOffline    EventType = "user.offline"
	EventReactionAdd    EventType = "reaction.add"
	EventReactionRemove EventType = "reaction.remove"
)

// Event represents a real-time event
type Event struct {
	Type      EventType   `json:"type"`
	ChatID    string      `json:"chat_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, chatID, userID string, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		ChatID:    chatID,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// PublishEvent publishes an event to chat channel
func (c *Client) PublishEvent(ctx context.Context, event *Event) error {
	channel := fmt.Sprintf("chat:%s", event.ChatID)
	return c.Publish(ctx, channel, event)
}

// PublishToUser publishes an event to user's personal channel
func (c *Client) PublishToUser(ctx context.Context, userID string, event *Event) error {
	channel := fmt.Sprintf("user:%s", userID)
	return c.Publish(ctx, channel, event)
}

// PublishToUsers publishes an event to multiple users
func (c *Client) PublishToUsers(ctx context.Context, userIDs []string, event *Event) error {
	channels := make([]string, len(userIDs))
	for i, userID := range userIDs {
		channels[i] = fmt.Sprintf("user:%s", userID)
	}
	return c.Broadcast(ctx, channels, event)
}
