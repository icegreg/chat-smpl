package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/logger"
	"github.com/icegreg/chat-smpl/pkg/validator"
	"github.com/icegreg/chat-smpl/services/users/internal/model"
	"github.com/icegreg/chat-smpl/services/users/internal/repository"
	"github.com/icegreg/chat-smpl/services/users/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	userService service.UserService
	jwtManager  *jwt.Manager
}

func New(userService service.UserService, jwtManager *jwt.Manager) *Handler {
	return &Handler{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// Auth routes (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)

		// Protected auth routes
		r.Group(func(r chi.Router) {
			r.Use(h.AuthMiddleware)
			r.Get("/me", h.GetCurrentUser)
		})
	})

	// User routes (protected)
	r.Route("/api/users", func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/", h.ListUsers)
		r.Get("/{userGUID}", h.GetUser)
	})

	// Group routes (protected)
	r.Route("/api/groups", func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Get("/", h.ListGroups)
		r.Get("/{groupID}", h.GetGroup)
	})
}

// Auth handlers

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validator.Validate(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.userService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			h.respondError(w, http.StatusConflict, "USER_EXISTS", "User with this email already exists")
			return
		}
		logger.Error("failed to register user", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validator.Validate(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.userService.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
			return
		}
		logger.Error("failed to login", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validator.Validate(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	resp, err := h.userService.Refresh(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired refresh token")
			return
		}
		logger.Error("failed to refresh token", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validator.Validate(req); err != nil {
		h.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if err := h.userService.Logout(r.Context(), req); err != nil {
		logger.Error("failed to logout", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok || userID == uuid.Nil {
		h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		logger.Error("failed to get current user", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, user)
}

// User handlers

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if count < 1 || count > 100 {
		count = 20
	}

	resp, err := h.userService.List(r.Context(), page, count)
	if err != nil {
		logger.Error("failed to list users", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users":      resp.Data,
		"pagination": resp.Pagination,
	})
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userGUID := chi.URLParam(r, "userGUID")
	id, err := uuid.Parse(userGUID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid user ID format")
		return
	}

	user, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			h.respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		logger.Error("failed to get user", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, user)
}

// Group handlers (placeholder implementations)

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement when group service is ready
	h.respondJSON(w, http.StatusOK, []interface{}{})
}

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement when group service is ready
	h.respondError(w, http.StatusNotFound, "NOT_FOUND", "Group not found")
}

// Middleware

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}

		// Extract token from "Bearer <token>"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
			return
		}

		token := authHeader[7:]
		claims, err := h.jwtManager.ValidateAccessToken(token)
		if err != nil {
			if errors.Is(err, jwt.ErrExpiredToken) {
				h.respondError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Access token has expired")
				return
			}
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid access token")
			return
		}

		// Add claims to context
		ctx := r.Context()
		ctx = SetUserID(ctx, claims.UserID)
		ctx = SetUserRole(ctx, model.Role(claims.Role))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Response helpers

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			logger.Error("failed to encode response", zap.Error(err))
		}
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, errCode, message string) {
	h.respondJSON(w, status, model.NewErrorResponse(errCode, message))
}
