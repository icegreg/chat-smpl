package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_Register(t *testing.T) {
	SkipIfNotIntegration(t)

	email := fmt.Sprintf("register_test_%d@example.com", time.Now().UnixNano())

	registerReq := map[string]string{
		"email":        email,
		"username":     fmt.Sprintf("reguser_%d", time.Now().UnixNano()),
		"password":     "testpassword123",
		"display_name": "Register Test User",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/register", registerReq, "")

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["access_token"])
	assert.NotEmpty(t, result["refresh_token"])
}

func TestAuth_Register_DuplicateEmail(t *testing.T) {
	SkipIfNotIntegration(t)

	email := fmt.Sprintf("duplicate_test_%d@example.com", time.Now().UnixNano())

	registerReq := map[string]string{
		"email":    email,
		"username": fmt.Sprintf("dupuser1_%d", time.Now().UnixNano()),
		"password": "testpassword123",
	}

	// First registration should succeed
	resp, _ := doRequest(t, "POST", apiGatewayURL+"/api/auth/register", registerReq, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Second registration with same email should fail
	registerReq["username"] = fmt.Sprintf("dupuser2_%d", time.Now().UnixNano())
	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/register", registerReq, "")

	assert.Equal(t, http.StatusConflict, resp.StatusCode, "Response: %s", string(body))
}

func TestAuth_Login(t *testing.T) {
	SkipIfNotIntegration(t)

	// Create user first
	user := createTestUser(t, "login")

	// Test login
	loginReq := map[string]string{
		"email":    user.Email,
		"password": "testpassword123",
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/login", loginReq, "")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["access_token"])
	assert.NotEmpty(t, result["refresh_token"])
}

func TestAuth_Login_InvalidCredentials(t *testing.T) {
	SkipIfNotIntegration(t)

	loginReq := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword",
	}

	resp, _ := doRequest(t, "POST", apiGatewayURL+"/api/auth/login", loginReq, "")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_RefreshToken(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "refresh")

	refreshReq := map[string]string{
		"refresh_token": user.RefreshToken,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/refresh", refreshReq, "")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result["access_token"])
	assert.NotEmpty(t, result["refresh_token"])
}

func TestAuth_GetCurrentUser(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "me")

	resp, body := doRequest(t, "GET", apiGatewayURL+"/api/auth/me", nil, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	var result map[string]interface{}
	err := json.Unmarshal(body, &result)
	require.NoError(t, err)

	assert.Equal(t, user.Email, result["email"])
	assert.Equal(t, user.Username, result["username"])
}

func TestAuth_GetCurrentUser_Unauthorized(t *testing.T) {
	SkipIfNotIntegration(t)

	resp, _ := doRequest(t, "GET", apiGatewayURL+"/api/auth/me", nil, "invalid-token")

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuth_Logout(t *testing.T) {
	SkipIfNotIntegration(t)

	user := createTestUser(t, "logout")

	logoutReq := map[string]string{
		"refresh_token": user.RefreshToken,
	}

	resp, body := doRequest(t, "POST", apiGatewayURL+"/api/auth/logout", logoutReq, user.AccessToken)

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Response: %s", string(body))

	// Refresh token should no longer work
	refreshReq := map[string]string{
		"refresh_token": user.RefreshToken,
	}

	resp, _ = doRequest(t, "POST", apiGatewayURL+"/api/auth/refresh", refreshReq, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
