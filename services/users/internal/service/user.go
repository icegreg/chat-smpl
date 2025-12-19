package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/pkg/password"
	"github.com/icegreg/chat-smpl/services/users/internal/avatar"
	"github.com/icegreg/chat-smpl/services/users/internal/model"
	"github.com/icegreg/chat-smpl/services/users/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrAccessDenied       = errors.New("access denied")
)

// generateLocalAvatarURL returns the local URL for a user's avatar
func generateLocalAvatarURL(userID uuid.UUID) string {
	return fmt.Sprintf("/api/users/avatars/%s", avatar.GetFilename(userID))
}

type UserService interface {
	// Auth
	Register(ctx context.Context, req model.RegisterRequest) (*model.RegisterResponse, error)
	Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error)
	Refresh(ctx context.Context, req model.RefreshRequest) (*model.RefreshResponse, error)
	Logout(ctx context.Context, req model.LogoutRequest) error

	// Users
	Create(ctx context.Context, req model.CreateUserRequest) (*model.UserDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserDTO, error)
	List(ctx context.Context, page, count int) (*model.PaginatedResponse[model.UserDTO], error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) (*model.UserDTO, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role model.Role) (*model.UserDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ChangePassword(ctx context.Context, id uuid.UUID, req model.ChangePasswordRequest) error

	// Avatars
	GetAvatarPath(userID uuid.UUID) string
	RegenerateMissingAvatars(ctx context.Context) (int, error)
}

type userService struct {
	repo            repository.UserRepository
	jwtManager      *jwt.Manager
	avatarGenerator *avatar.Generator
}

func NewUserService(repo repository.UserRepository, jwtManager *jwt.Manager, avatarsPath string) UserService {
	return &userService{
		repo:            repo,
		jwtManager:      jwtManager,
		avatarGenerator: avatar.NewGenerator(avatarsPath),
	}
}

func (s *userService) Register(ctx context.Context, req model.RegisterRequest) (*model.RegisterResponse, error) {
	// Create user with default role
	userDTO, err := s.Create(ctx, model.CreateUserRequest{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
		Role:        model.RoleUser,
	})
	if err != nil {
		return nil, err
	}

	// Generate tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(userDTO.ID, userDTO.Username, userDTO.Email, string(userDTO.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Store refresh token
	refreshToken := &model.RefreshToken{
		UserID:    userDTO.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: tokenPair.ExpiresAt.AddDate(0, 0, 7),
	}
	if err := s.repo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &model.RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		User:         *userDTO,
	}, nil
}

func (s *userService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if err := password.Verify(req.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Store refresh token
	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: tokenPair.ExpiresAt.AddDate(0, 0, 7), // Refresh token expires in 7 days
	}
	if err := s.repo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &model.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		User:         user.ToDTO(),
	}, nil
}

func (s *userService) Refresh(ctx context.Context, req model.RefreshRequest) (*model.RefreshResponse, error) {
	// Validate refresh token format
	claims, err := s.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check if token exists in database
	storedToken, err := s.repo.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, repository.ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Get user
	user, err := s.repo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Delete old refresh token
	if err := s.repo.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	// Generate new token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, user.Email, string(user.Role))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token pair: %w", err)
	}

	// Store new refresh token
	newRefreshToken := &model.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: tokenPair.ExpiresAt.AddDate(0, 0, 7),
	}
	if err := s.repo.CreateRefreshToken(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	_ = claims // используется для валидации

	return &model.RefreshResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	}, nil
}

func (s *userService) Logout(ctx context.Context, req model.LogoutRequest) error {
	if err := s.repo.DeleteRefreshToken(ctx, req.RefreshToken); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

func (s *userService) Create(ctx context.Context, req model.CreateUserRequest) (*model.UserDTO, error) {
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: hashedPassword,
		Role:         req.Role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate local avatar file
	if err := s.avatarGenerator.Generate(user.ID); err != nil {
		fmt.Printf("warning: failed to generate avatar: %v\n", err)
	}

	// Set avatar URL to local path
	avatarURL := generateLocalAvatarURL(user.ID)
	user.AvatarURL = &avatarURL
	if err := s.repo.Update(ctx, user); err != nil {
		fmt.Printf("warning: failed to update avatar URL: %v\n", err)
	}

	dto := user.ToDTO()
	return &dto, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*model.UserDTO, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	dto := user.ToDTO()
	return &dto, nil
}

func (s *userService) List(ctx context.Context, page, count int) (*model.PaginatedResponse[model.UserDTO], error) {
	users, total, err := s.repo.List(ctx, page, count)
	if err != nil {
		return nil, err
	}

	dtos := make([]model.UserDTO, len(users))
	for i, user := range users {
		dtos[i] = user.ToDTO()
	}

	totalPages := total / count
	if total%count > 0 {
		totalPages++
	}

	return &model.PaginatedResponse[model.UserDTO]{
		Data: dtos,
		Pagination: model.Pagination{
			Page:       page,
			Count:      count,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) (*model.UserDTO, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Role != nil {
		user.Role = *req.Role
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	dto := user.ToDTO()
	return &dto, nil
}

func (s *userService) UpdateRole(ctx context.Context, id uuid.UUID, role model.Role) (*model.UserDTO, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	user.Role = role

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	dto := user.ToDTO()
	return &dto, nil
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *userService) ChangePassword(ctx context.Context, id uuid.UUID, req model.ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := password.Verify(req.OldPassword, user.PasswordHash); err != nil {
		return ErrInvalidCredentials
	}

	hashedPassword, err := password.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = hashedPassword
	return s.repo.Update(ctx, user)
}

// GetAvatarPath returns the filesystem path to a user's avatar
func (s *userService) GetAvatarPath(userID uuid.UUID) string {
	return s.avatarGenerator.GetPath(userID)
}

// RegenerateMissingAvatars checks all users and regenerates avatars for those missing files
func (s *userService) RegenerateMissingAvatars(ctx context.Context) (int, error) {
	// Get all users
	users, _, err := s.repo.List(ctx, 1, 10000) // Get all users
	if err != nil {
		return 0, fmt.Errorf("failed to list users: %w", err)
	}

	regenerated := 0
	for _, user := range users {
		// Check if user has avatar URL but file doesn't exist
		if user.AvatarURL != nil && *user.AvatarURL != "" {
			if !s.avatarGenerator.Exists(user.ID) {
				// Regenerate avatar
				if err := s.avatarGenerator.Generate(user.ID); err != nil {
					fmt.Printf("warning: failed to regenerate avatar for user %s: %v\n", user.ID, err)
					continue
				}

				// Update avatar URL to local path
				avatarURL := generateLocalAvatarURL(user.ID)
				user.AvatarURL = &avatarURL
				if err := s.repo.Update(ctx, &user); err != nil {
					fmt.Printf("warning: failed to update avatar URL for user %s: %v\n", user.ID, err)
				}

				regenerated++
				fmt.Printf("Regenerated avatar for user %s (%s)\n", user.Username, user.ID)
			}
		} else {
			// User doesn't have avatar at all - generate one
			if err := s.avatarGenerator.Generate(user.ID); err != nil {
				fmt.Printf("warning: failed to generate avatar for user %s: %v\n", user.ID, err)
				continue
			}

			avatarURL := generateLocalAvatarURL(user.ID)
			user.AvatarURL = &avatarURL
			if err := s.repo.Update(ctx, &user); err != nil {
				fmt.Printf("warning: failed to update avatar URL for user %s: %v\n", user.ID, err)
			}

			regenerated++
			fmt.Printf("Generated avatar for user %s (%s)\n", user.Username, user.ID)
		}
	}

	return regenerated, nil
}
