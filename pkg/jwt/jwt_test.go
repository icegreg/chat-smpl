package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	secret := "test-secret"
	cfg := DefaultConfig(secret)

	assert.Equal(t, secret, cfg.Secret)
	assert.Equal(t, 15*time.Minute, cfg.AccessTokenTTL)
	assert.Equal(t, 7*24*time.Hour, cfg.RefreshTokenTTL)
	assert.Equal(t, "chat-smpl", cfg.Issuer)
}

func TestNewManager(t *testing.T) {
	cfg := DefaultConfig("test-secret")
	manager := NewManager(cfg)

	assert.NotNil(t, manager)
}

func TestGenerateAccessToken(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))
	userID := uuid.New()

	token, expiresAt, err := manager.GenerateAccessToken(userID, "testuser", "test@example.com", "user")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
}

func TestGenerateRefreshToken(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))

	token, expiresAt, err := manager.GenerateRefreshToken()

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
}

func TestGenerateTokenPair(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))
	userID := uuid.New()

	pair, err := manager.GenerateTokenPair(userID, "testuser", "test@example.com", "user")

	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.True(t, pair.ExpiresAt.After(time.Now()))
}

func TestValidateAccessToken(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))
	userID := uuid.New()
	username := "testuser"
	email := "test@example.com"
	role := "user"

	token, _, err := manager.GenerateAccessToken(userID, username, email, role)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))

	_, err := manager.ValidateAccessToken("invalid-token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateAccessToken_Expired(t *testing.T) {
	cfg := Config{
		Secret:          "test-secret",
		AccessTokenTTL:  -1 * time.Hour, // Already expired
		RefreshTokenTTL: 7 * 24 * time.Hour,
		Issuer:          "chat-smpl",
	}
	manager := NewManager(cfg)
	userID := uuid.New()

	token, _, err := manager.GenerateAccessToken(userID, "testuser", "test@example.com", "user")
	require.NoError(t, err)

	_, err = manager.ValidateAccessToken(token)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	manager1 := NewManager(DefaultConfig("secret1"))
	manager2 := NewManager(DefaultConfig("secret2"))
	userID := uuid.New()

	token, _, err := manager1.GenerateAccessToken(userID, "testuser", "test@example.com", "user")
	require.NoError(t, err)

	_, err = manager2.ValidateAccessToken(token)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateRefreshToken(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))

	token, _, err := manager.GenerateRefreshToken()
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.ID)
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	manager := NewManager(DefaultConfig("test-secret"))

	_, err := manager.ValidateRefreshToken("invalid-token")
	assert.ErrorIs(t, err, ErrInvalidToken)
}
