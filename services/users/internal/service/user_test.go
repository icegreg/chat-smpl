package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/icegreg/chat-smpl/pkg/jwt"
	"github.com/icegreg/chat-smpl/services/users/internal/model"
	"github.com/icegreg/chat-smpl/services/users/internal/repository"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	if args.Error(0) == nil {
		user.ID = uuid.New()
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()
	}
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, page, count int) ([]model.User, int, error) {
	args := m.Called(ctx, page, count)
	return args.Get(0).([]model.User), args.Int(1), args.Error(2)
}

func (m *MockUserRepository) Search(ctx context.Context, query string, page, count int) ([]model.User, int, error) {
	args := m.Called(ctx, query, page, count)
	return args.Get(0).([]model.User), args.Int(1), args.Error(2)
}

func (m *MockUserRepository) GetByExtension(ctx context.Context, extension string) (*model.User, error) {
	args := m.Called(ctx, extension)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) GetNextExtension(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockUserRepository) AssignExtension(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) CreateRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserRepository) GetRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.RefreshToken), args.Error(1)
}

func (m *MockUserRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockUserRepository) DeleteExpiredTokens(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestUserService_Create(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()
	req := model.CreateUserRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
		Role:     model.RoleUser,
	}

	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.User")).Return(nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

	user, err := svc.Create(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, req.Username, user.Username)
	assert.Equal(t, req.Email, user.Email)
	assert.Equal(t, req.Role, user.Role)

	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_Success(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()
	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	existingUser := &model.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Role:         model.RoleUser,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockRepo.On("GetByEmail", ctx, "test@example.com").Return(existingUser, nil)
	mockRepo.On("CreateRefreshToken", ctx, mock.AnythingOfType("*model.RefreshToken")).Return(nil)

	req := model.LoginRequest{
		Email:    "test@example.com",
		Password: password,
	}

	resp, err := svc.Login(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, existingUser.Username, resp.User.Username)

	mockRepo.AssertExpectations(t)
}

func TestUserService_Login_InvalidCredentials(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()

	mockRepo.On("GetByEmail", ctx, "test@example.com").Return(nil, repository.ErrUserNotFound)

	req := model.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	resp, err := svc.Login(ctx, req)

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, resp)

	mockRepo.AssertExpectations(t)
}

func TestUserService_GetByID(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()
	userID := uuid.New()

	existingUser := &model.User{
		ID:        userID,
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      model.RoleUser,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockRepo.On("GetByID", ctx, userID).Return(existingUser, nil)

	user, err := svc.GetByID(ctx, userID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "testuser", user.Username)

	mockRepo.AssertExpectations(t)
}

func TestUserService_List(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()

	users := []model.User{
		{ID: uuid.New(), Username: "user1", Email: "user1@example.com", Role: model.RoleUser},
		{ID: uuid.New(), Username: "user2", Email: "user2@example.com", Role: model.RoleModerator},
	}

	mockRepo.On("List", ctx, 1, 20).Return(users, 2, nil)

	resp, err := svc.List(ctx, 1, 20)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 2, resp.Pagination.Total)

	mockRepo.AssertExpectations(t)
}

func TestUserService_UpdateRole(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()
	userID := uuid.New()

	existingUser := &model.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     model.RoleUser,
	}

	mockRepo.On("GetByID", ctx, userID).Return(existingUser, nil)
	mockRepo.On("Update", ctx, mock.AnythingOfType("*model.User")).Return(nil)

	user, err := svc.UpdateRole(ctx, userID, model.RoleModerator)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, model.RoleModerator, user.Role)

	mockRepo.AssertExpectations(t)
}

func TestUserService_Delete(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtManager := jwt.NewManager(jwt.DefaultConfig("test-secret"))
	svc := NewUserService(mockRepo, jwtManager, "/tmp/avatars")

	ctx := context.Background()
	userID := uuid.New()

	mockRepo.On("Delete", ctx, userID).Return(nil)

	err := svc.Delete(ctx, userID)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
