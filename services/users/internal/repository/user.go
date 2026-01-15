package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/icegreg/chat-smpl/services/users/internal/model"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrTokenNotFound      = errors.New("refresh token not found")
	ErrGroupNotFound      = errors.New("group not found")
	ErrGroupAlreadyExists = errors.New("group already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByExtension(ctx context.Context, extension string) (*model.User, error)
	List(ctx context.Context, page, count int) ([]model.User, int, error)
	Search(ctx context.Context, query string, page, count int) ([]model.User, int, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error

	// Extension management
	GetNextExtension(ctx context.Context) (string, error)
	AssignExtension(ctx context.Context, userID uuid.UUID) (string, error)

	// Refresh tokens
	CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error
	GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteExpiredTokens(ctx context.Context) error
}

type userRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO con_test.users (id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Auto-assign extension and SIP password for non-guest users
	if user.Extension == nil && user.Role != model.RoleGuest {
		ext, err := r.GetNextExtension(ctx)
		if err != nil {
			return fmt.Errorf("failed to get next extension: %w", err)
		}
		user.Extension = &ext

		// Generate SIP password
		sipPass, err := r.generateSIPPassword(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate SIP password: %w", err)
		}
		user.SIPPassword = &sipPass
	}

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.DisplayName,
		user.AvatarURL,
		user.Extension,
		user.SIPPassword,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepository) generateSIPPassword(ctx context.Context) (string, error) {
	query := `SELECT con_test.generate_sip_password()`
	var password string
	err := r.pool.QueryRow(ctx, query).Scan(&password)
	if err != nil {
		return "", err
	}
	return password, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		WHERE id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Extension,
		&user.SIPPassword,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		WHERE email = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Extension,
		&user.SIPPassword,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		WHERE username = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Extension,
		&user.SIPPassword,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByExtension(ctx context.Context, extension string) (*model.User, error) {
	query := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		WHERE extension = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, extension).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.AvatarURL,
		&user.Extension,
		&user.SIPPassword,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by extension: %w", err)
	}

	return &user, nil
}

func (r *userRepository) List(ctx context.Context, page, count int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}

	offset := (page - 1) * count

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM con_test.users`
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users
	query := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.DisplayName,
			&user.AvatarURL,
			&user.Extension,
			&user.SIPPassword,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (r *userRepository) Search(ctx context.Context, query string, page, count int) ([]model.User, int, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 || count > 100 {
		count = 20
	}

	offset := (page - 1) * count
	searchPattern := "%" + query + "%"

	// Get total count with search filter
	var total int
	countQuery := `
		SELECT COUNT(*) FROM con_test.users
		WHERE username ILIKE $1
		   OR display_name ILIKE $1
	`
	if err := r.pool.QueryRow(ctx, countQuery, searchPattern).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get users with search filter
	searchQuery := `
		SELECT id, username, email, display_name, avatar_url, extension, sip_password, password_hash, role, created_at, updated_at
		FROM con_test.users
		WHERE username ILIKE $1
		   OR display_name ILIKE $1
		ORDER BY
			CASE
				WHEN username ILIKE $2 THEN 0
				WHEN display_name ILIKE $2 THEN 1
				ELSE 2
			END,
			display_name, username
		LIMIT $3 OFFSET $4
	`
	// $2 is exact prefix match for better ranking
	prefixPattern := query + "%"

	rows, err := r.pool.Query(ctx, searchQuery, searchPattern, prefixPattern, count, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.DisplayName,
			&user.AvatarURL,
			&user.Extension,
			&user.SIPPassword,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE con_test.users
		SET username = $2, email = $3, display_name = $4, avatar_url = $5, extension = $6, sip_password = $7, password_hash = $8, role = $9, updated_at = $10
		WHERE id = $1
	`

	user.UpdatedAt = time.Now()

	result, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.DisplayName,
		user.AvatarURL,
		user.Extension,
		user.SIPPassword,
		user.PasswordHash,
		user.Role,
		user.UpdatedAt,
	)

	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *userRepository) GetNextExtension(ctx context.Context) (string, error) {
	query := `SELECT con_test.get_next_extension()`

	var ext string
	err := r.pool.QueryRow(ctx, query).Scan(&ext)
	if err != nil {
		return "", fmt.Errorf("failed to get next extension: %w", err)
	}

	return ext, nil
}

func (r *userRepository) AssignExtension(ctx context.Context, userID uuid.UUID) (string, error) {
	ext, err := r.GetNextExtension(ctx)
	if err != nil {
		return "", err
	}

	query := `UPDATE con_test.users SET extension = $2, updated_at = $3 WHERE id = $1 AND extension IS NULL`

	result, err := r.pool.Exec(ctx, query, userID, ext, time.Now())
	if err != nil {
		return "", fmt.Errorf("failed to assign extension: %w", err)
	}

	if result.RowsAffected() == 0 {
		// User already has extension or not found
		return "", ErrUserNotFound
	}

	return ext, nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM con_test.users WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *userRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO con_test.refresh_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	token.ID = uuid.New()
	token.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.Token,
		token.ExpiresAt,
		token.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *userRepository) GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM con_test.refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var rt model.RefreshToken
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.Token,
		&rt.ExpiresAt,
		&rt.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &rt, nil
}

func (r *userRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM con_test.refresh_tokens WHERE token = $1`

	_, err := r.pool.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

func (r *userRepository) DeleteExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM con_test.refresh_tokens WHERE expires_at < NOW()`

	_, err := r.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsRune(s, substr))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
