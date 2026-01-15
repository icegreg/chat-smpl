package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Test configuration
var (
	apiGatewayURL  = getEnv("API_GATEWAY_URL", "http://localhost:8080")
	usersServiceURL = getEnv("USERS_SERVICE_URL", "http://localhost:8081")
	filesServiceURL = getEnv("FILES_SERVICE_URL", "http://localhost:8082")
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTTP client with timeout
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// TestUser represents a test user
type TestUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TestChat represents a test chat
type TestChat struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// TestMessage represents a test message
type TestMessage struct {
	ID       string `json:"id"`
	ChatID   string `json:"chat_id"`
	SenderID string `json:"sender_id"`
	Content  string `json:"content"`
}

// Helper function to make HTTP requests
func doRequest(t *testing.T, method, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return resp, respBody
}

// Helper to create a test user
func createTestUser(t *testing.T, suffix string) *TestUser {
	t.Helper()

	email := fmt.Sprintf("test%s_%d@example.com", suffix, time.Now().UnixNano())
	username := fmt.Sprintf("testuser%s_%d", suffix, time.Now().UnixNano())

	registerReq := map[string]string{
		"email":        email,
		"username":     username,
		"password":     "testpassword123",
		"display_name": "Test User " + suffix,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/register", registerReq, "")

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register user: %s, status: %d", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse register response: %v", err)
	}

	// Get user info
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/auth/me", nil, result.AccessToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get user info: %s", string(body))
	}

	var user TestUser
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("Failed to parse user info: %v", err)
	}

	user.AccessToken = result.AccessToken
	user.RefreshToken = result.RefreshToken

	return &user
}

// Helper to create a test chat
func createTestChat(t *testing.T, user *TestUser, chatType, name string, participantIDs []string) *TestChat {
	t.Helper()

	chatReq := map[string]interface{}{
		"type":           chatType,
		"name":           name,
		"description":    "Test chat description",
		"participant_ids": participantIDs,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats", chatReq, user.AccessToken)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create chat: %s, status: %d", string(body), resp.StatusCode)
	}

	var chat TestChat
	if err := json.Unmarshal(body, &chat); err != nil {
		t.Fatalf("Failed to parse chat response: %v", err)
	}

	return &chat
}

// Helper to send a test message
func sendTestMessage(t *testing.T, user *TestUser, chatID, content string) *TestMessage {
	t.Helper()

	msgReq := map[string]string{
		"content": content,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/chats/"+chatID+"/messages", msgReq, user.AccessToken)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to send message: %s, status: %d", string(body), resp.StatusCode)
	}

	var msg TestMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("Failed to parse message response: %v", err)
	}

	return &msg
}

// SkipIfNotIntegration skips test if not running integration tests
func SkipIfNotIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}
}

// assertErrorResponse checks that response has expected error status
func assertErrorResponse(t *testing.T, resp *http.Response, body []byte, expectedStatus int) {
	t.Helper()
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, resp.StatusCode, string(body))
	}
}

// createTestUserWithPassword creates a test user with specific password
func createTestUserWithPassword(t *testing.T, suffix, password string) *TestUser {
	t.Helper()

	email := fmt.Sprintf("test%s_%d@example.com", suffix, time.Now().UnixNano())
	username := fmt.Sprintf("testuser%s_%d", suffix, time.Now().UnixNano())

	registerReq := map[string]string{
		"email":        email,
		"username":     username,
		"password":     password,
		"display_name": "Test User " + suffix,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/register", registerReq, "")

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to register user: %s, status: %d", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse register response: %v", err)
	}

	// Get user info
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/auth/me", nil, result.AccessToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get user info: %s", string(body))
	}

	var user TestUser
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("Failed to parse user info: %v", err)
	}

	user.AccessToken = result.AccessToken
	user.RefreshToken = result.RefreshToken
	user.Email = email

	return &user
}

// loginUser logs in a user and returns TestUser with tokens
func loginUser(t *testing.T, email, password string) *TestUser {
	t.Helper()

	loginReq := map[string]string{
		"email":    email,
		"password": password,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/login", loginReq, "")

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to login: %s, status: %d", string(body), resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse login response: %v", err)
	}

	// Get user info
	resp, body = doRequest(t, "GET", apiGatewayURL+"/api/auth/me", nil, result.AccessToken)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get user info: %s", string(body))
	}

	var user TestUser
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("Failed to parse user info: %v", err)
	}

	user.AccessToken = result.AccessToken
	user.RefreshToken = result.RefreshToken
	user.Email = email

	return &user
}

// deleteTestChat deletes a chat
func deleteTestChat(t *testing.T, user *TestUser, chatID string) {
	t.Helper()
	resp, _ := doRequest(t, "DELETE", apiGatewayURL+"/api/chats/"+chatID, nil, user.AccessToken)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Logf("Warning: failed to delete chat %s, status: %d", chatID, resp.StatusCode)
	}
}
