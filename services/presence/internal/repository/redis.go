package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes
	keyStatusPrefix      = "presence:status:"      // presence:status:{userId} -> status (available, busy, away, dnd)
	keyConnectionsPrefix = "presence:connections:" // presence:connections:{userId} -> set of connection IDs
	keyLastSeenPrefix    = "presence:lastseen:"    // presence:lastseen:{userId} -> timestamp

	// TTLs
	lastSeenTTL = 7 * 24 * time.Hour // Keep last seen for 7 days
)

// UserStatus represents user's manually set status
type UserStatus string

const (
	StatusAvailable UserStatus = "available"
	StatusBusy      UserStatus = "busy"
	StatusAway      UserStatus = "away"
	StatusDND       UserStatus = "dnd"
)

// PresenceInfo contains full presence information
type PresenceInfo struct {
	UserID          string
	Status          UserStatus
	IsOnline        bool
	ConnectionCount int
	LastSeenAt      time.Time
}

// Repository handles presence data in Redis
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new presence repository
func NewRepository(client *redis.Client) *Repository {
	return &Repository{client: client}
}

// SetStatus sets user's status
func (r *Repository) SetStatus(ctx context.Context, userID string, status UserStatus) error {
	key := keyStatusPrefix + userID
	return r.client.Set(ctx, key, string(status), 0).Err() // No expiration for status
}

// GetStatus gets user's status
func (r *Repository) GetStatus(ctx context.Context, userID string) (UserStatus, error) {
	key := keyStatusPrefix + userID
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return StatusAvailable, nil // Default status
	}
	if err != nil {
		return "", err
	}
	return UserStatus(val), nil
}

// AddConnection adds a connection for a user
func (r *Repository) AddConnection(ctx context.Context, userID, connectionID string) error {
	key := keyConnectionsPrefix + userID
	if err := r.client.SAdd(ctx, key, connectionID).Err(); err != nil {
		return err
	}
	// Update last seen
	return r.updateLastSeen(ctx, userID)
}

// RemoveConnection removes a connection for a user
func (r *Repository) RemoveConnection(ctx context.Context, userID, connectionID string) error {
	key := keyConnectionsPrefix + userID
	if err := r.client.SRem(ctx, key, connectionID).Err(); err != nil {
		return err
	}
	// Update last seen
	return r.updateLastSeen(ctx, userID)
}

// GetConnectionCount returns the number of active connections for a user
func (r *Repository) GetConnectionCount(ctx context.Context, userID string) (int, error) {
	key := keyConnectionsPrefix + userID
	count, err := r.client.SCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// IsOnline checks if user has any active connections
func (r *Repository) IsOnline(ctx context.Context, userID string) (bool, error) {
	count, err := r.GetConnectionCount(ctx, userID)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetLastSeen returns when user was last seen
func (r *Repository) GetLastSeen(ctx context.Context, userID string) (time.Time, error) {
	key := keyLastSeenPrefix + userID
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	ts, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

func (r *Repository) updateLastSeen(ctx context.Context, userID string) error {
	key := keyLastSeenPrefix + userID
	return r.client.Set(ctx, key, strconv.FormatInt(time.Now().Unix(), 10), lastSeenTTL).Err()
}

// GetPresence returns full presence info for a user
func (r *Repository) GetPresence(ctx context.Context, userID string) (*PresenceInfo, error) {
	status, err := r.GetStatus(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	count, err := r.GetConnectionCount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection count: %w", err)
	}

	lastSeen, err := r.GetLastSeen(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last seen: %w", err)
	}

	return &PresenceInfo{
		UserID:          userID,
		Status:          status,
		IsOnline:        count > 0,
		ConnectionCount: count,
		LastSeenAt:      lastSeen,
	}, nil
}

// GetPresencesBatch returns presence info for multiple users
func (r *Repository) GetPresencesBatch(ctx context.Context, userIDs []string) ([]*PresenceInfo, error) {
	results := make([]*PresenceInfo, 0, len(userIDs))

	// Use pipeline for efficiency
	pipe := r.client.Pipeline()

	// Queue all commands
	statusCmds := make([]*redis.StringCmd, len(userIDs))
	countCmds := make([]*redis.IntCmd, len(userIDs))
	lastSeenCmds := make([]*redis.StringCmd, len(userIDs))

	for i, userID := range userIDs {
		statusCmds[i] = pipe.Get(ctx, keyStatusPrefix+userID)
		countCmds[i] = pipe.SCard(ctx, keyConnectionsPrefix+userID)
		lastSeenCmds[i] = pipe.Get(ctx, keyLastSeenPrefix+userID)
	}

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("pipeline exec failed: %w", err)
	}

	// Collect results
	for i, userID := range userIDs {
		status := StatusAvailable
		if val, err := statusCmds[i].Result(); err == nil {
			status = UserStatus(val)
		}

		count := int64(0)
		if val, err := countCmds[i].Result(); err == nil {
			count = val
		}

		var lastSeen time.Time
		if val, err := lastSeenCmds[i].Result(); err == nil {
			if ts, err := strconv.ParseInt(val, 10, 64); err == nil {
				lastSeen = time.Unix(ts, 0)
			}
		}

		results = append(results, &PresenceInfo{
			UserID:          userID,
			Status:          status,
			IsOnline:        count > 0,
			ConnectionCount: int(count),
			LastSeenAt:      lastSeen,
		})
	}

	return results, nil
}
