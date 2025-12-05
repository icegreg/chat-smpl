package model

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner     Role = "owner"
	RoleModerator Role = "moderator"
	RoleUser      Role = "user"
	RoleGuest     Role = "guest"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleModerator, RoleUser, RoleGuest:
		return true
	}
	return false
}

func (r Role) CanWrite() bool {
	return r != RoleGuest
}

func (r Role) CanModerate() bool {
	return r == RoleOwner || r == RoleModerator
}

func (r Role) IsAdmin() bool {
	return r == RoleOwner
}

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	DisplayName  *string   `json:"display_name,omitempty" db:"display_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Role         Role      `json:"role" db:"role"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Group struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type GroupMember struct {
	ID       uuid.UUID `json:"id" db:"id"`
	GroupID  uuid.UUID `json:"group_id" db:"group_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}
