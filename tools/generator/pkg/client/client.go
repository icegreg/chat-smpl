package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	User         User   `json:"user"`
}

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type CreateChatRequest struct {
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	ParticipantIDs []string `json:"participant_ids"`
}

type Chat struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	ChatType  interface{} `json:"chat_type"` // Can be string or int (proto enum)
	CreatedBy string      `json:"created_by"`
}

type SendMessageRequest struct {
	Content string `json:"content"`
}

type Message struct {
	ID       string `json:"id"`
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Content  string `json:"content"`
}

type ListMessagesResponse struct {
	Messages   []Message  `json:"messages"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	Count      int `json:"count"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func (c *Client) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	return doPost[AuthResponse](c, ctx, "/api/auth/register", req, "")
}

func (c *Client) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	return doPost[AuthResponse](c, ctx, "/api/auth/login", req, "")
}

func (c *Client) CreateChat(ctx context.Context, token string, req CreateChatRequest) (*Chat, error) {
	return doPost[Chat](c, ctx, "/api/chats", req, token)
}

func (c *Client) SendMessage(ctx context.Context, token, chatID string, req SendMessageRequest) (*Message, error) {
	return doPost[Message](c, ctx, fmt.Sprintf("/api/chats/%s/messages", chatID), req, token)
}

func (c *Client) GetMessages(ctx context.Context, token, chatID string, page, count int) (*ListMessagesResponse, error) {
	url := fmt.Sprintf("/api/chats/%s/messages?page=%d&count=%d", chatID, page, count)
	return doGet[ListMessagesResponse](c, ctx, url, token)
}

func doPost[T any](c *Client, ctx context.Context, path string, body interface{}, token string) (*T, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result T
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func doGet[T any](c *Client, ctx context.Context, path string, token string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result T
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
