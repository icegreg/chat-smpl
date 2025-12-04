package service

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/icegreg/chat-smpl/services/files/internal/model"
	"github.com/icegreg/chat-smpl/services/files/internal/repository"
)

// MockRepository implements repository.FileRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateFile(ctx context.Context, file *model.File) error {
	args := m.Called(ctx, file)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	file.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetFile(ctx context.Context, id uuid.UUID) (*model.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.File), args.Error(1)
}

func (m *MockRepository) UpdateFileStatus(ctx context.Context, id uuid.UUID, status model.FileStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockRepository) DeleteFile(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateFileLink(ctx context.Context, link *model.FileLink) error {
	args := m.Called(ctx, link)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	link.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetFileLink(ctx context.Context, id uuid.UUID) (*model.FileLink, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FileLink), args.Error(1)
}

func (m *MockRepository) GetFileLinkByFileID(ctx context.Context, fileID uuid.UUID) (*model.FileLink, error) {
	args := m.Called(ctx, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FileLink), args.Error(1)
}

func (m *MockRepository) DeleteFileLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) SoftDeleteFileLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateFileLinkPermission(ctx context.Context, perm *model.FileLinkPermission) error {
	args := m.Called(ctx, perm)
	return args.Error(0)
}

func (m *MockRepository) GetFileLinkPermission(ctx context.Context, fileLinkID, userID uuid.UUID) (*model.FileLinkPermission, error) {
	args := m.Called(ctx, fileLinkID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FileLinkPermission), args.Error(1)
}

func (m *MockRepository) CreatePermissionsForParticipants(ctx context.Context, fileLinkID uuid.UUID, participantIDs []uuid.UUID, uploaderID uuid.UUID) error {
	args := m.Called(ctx, fileLinkID, participantIDs, uploaderID)
	return args.Error(0)
}

func (m *MockRepository) CreateShareLink(ctx context.Context, link *model.FileShareLink) error {
	args := m.Called(ctx, link)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	link.ID = uuid.New()
	link.Token = "test-token-123"
	return nil
}

func (m *MockRepository) GetShareLinkByToken(ctx context.Context, token string) (*model.FileShareLink, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FileShareLink), args.Error(1)
}

func (m *MockRepository) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeactivateShareLink(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) CreateMessageAttachment(ctx context.Context, attachment *model.MessageFileAttachment) error {
	args := m.Called(ctx, attachment)
	return args.Error(0)
}

func (m *MockRepository) GetMessageAttachments(ctx context.Context, messageID uuid.UUID) ([]model.MessageFileAttachment, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.MessageFileAttachment), args.Error(1)
}

// MockStorage implements storage.Storage for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Save(filename string, reader io.Reader) (string, error) {
	args := m.Called(filename, reader)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Get(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) Delete(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockStorage) Exists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func TestFileService_Upload(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	filename := "test.txt"
	contentType := "text/plain"
	content := []byte("test content")
	size := int64(len(content))

	mockStorage.On("Save", filename, mock.Anything).Return("2024/01/01/abc123_test.txt", nil)
	mockRepo.On("CreateFile", ctx, mock.AnythingOfType("*model.File")).Return(nil)
	mockRepo.On("CreateFileLink", ctx, mock.AnythingOfType("*model.FileLink")).Return(nil)
	mockRepo.On("CreateFileLinkPermission", ctx, mock.AnythingOfType("*model.FileLinkPermission")).Return(nil)

	reader := bytes.NewReader(content)
	response, err := svc.Upload(ctx, filename, contentType, size, reader, userID)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, filename, response.OriginalFilename)
	assert.Equal(t, contentType, response.ContentType)
	assert.Equal(t, size, response.Size)

	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestFileService_Download(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID:  linkID,
		UserID:      userID,
		CanView:     true,
		CanDownload: true,
	}

	file := &model.File{
		ID:               fileID,
		Filename:         "abc123_test.txt",
		OriginalFilename: "test.txt",
		ContentType:      "text/plain",
		Size:             12,
		FilePath:         "2024/01/01/abc123_test.txt",
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)
	mockRepo.On("GetFile", ctx, fileID).Return(file, nil)
	mockStorage.On("Get", file.FilePath).Return(io.NopCloser(bytes.NewReader([]byte("test content"))), nil)

	reader, returnedFile, err := svc.Download(ctx, linkID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, reader)
	assert.NotNil(t, returnedFile)
	assert.Equal(t, file.OriginalFilename, returnedFile.OriginalFilename)

	reader.Close()
	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestFileService_Download_NoPermission(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID:  linkID,
		UserID:      userID,
		CanView:     true,
		CanDownload: false, // No download permission
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)

	reader, file, err := svc.Download(ctx, linkID, userID)

	assert.Error(t, err)
	assert.Equal(t, repository.ErrAccessDenied, err)
	assert.Nil(t, reader)
	assert.Nil(t, file)

	mockRepo.AssertExpectations(t)
}

func TestFileService_Delete(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID:  linkID,
		UserID:      userID,
		CanView:     true,
		CanDownload: true,
		CanDelete:   true, // Has delete permission
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)
	mockRepo.On("SoftDeleteFileLink", ctx, linkID).Return(nil)

	err := svc.Delete(ctx, linkID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_Delete_NoPermission(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID:  linkID,
		UserID:      userID,
		CanView:     true,
		CanDownload: true,
		CanDelete:   false, // No delete permission
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)

	err := svc.Delete(ctx, linkID, userID)

	assert.Error(t, err)
	assert.Equal(t, repository.ErrAccessDenied, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_CreateShareLink(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID: linkID,
		UserID:     userID,
		CanView:    true,
	}

	mockRepo.On("GetFileLinkByFileID", ctx, fileID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)
	mockRepo.On("CreateShareLink", ctx, mock.AnythingOfType("*model.FileShareLink")).Return(nil)

	maxDownloads := 10
	req := model.CreateShareLinkRequest{
		MaxDownloads: &maxDownloads,
	}

	shareLink, err := svc.CreateShareLink(ctx, fileID, userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, shareLink)
	assert.NotEmpty(t, shareLink.Token)
	assert.Contains(t, shareLink.URL, "/share/")

	mockRepo.AssertExpectations(t)
}

func TestFileService_GetFileInfo(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:        linkID,
		FileID:    fileID,
		IsDeleted: false,
	}

	permission := &model.FileLinkPermission{
		FileLinkID: linkID,
		UserID:     userID,
		CanView:    true,
	}

	file := &model.File{
		ID:               fileID,
		Filename:         "abc123_test.txt",
		OriginalFilename: "test.txt",
		ContentType:      "text/plain",
		Size:             12,
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)
	mockRepo.On("GetFile", ctx, fileID).Return(file, nil)

	fileDTO, err := svc.GetFileInfo(ctx, linkID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, fileDTO)
	assert.Equal(t, file.OriginalFilename, fileDTO.OriginalFilename)
	assert.Equal(t, file.ContentType, fileDTO.ContentType)
	assert.Equal(t, file.Size, fileDTO.Size)

	mockRepo.AssertExpectations(t)
}
