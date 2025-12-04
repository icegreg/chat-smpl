package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/services/users/internal/model"
)

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	userRoleKey contextKey = "user_role"
)

func SetUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

func SetUserRole(ctx context.Context, role model.Role) context.Context {
	return context.WithValue(ctx, userRoleKey, role)
}

func GetUserRole(ctx context.Context) (model.Role, bool) {
	role, ok := ctx.Value(userRoleKey).(model.Role)
	return role, ok
}
