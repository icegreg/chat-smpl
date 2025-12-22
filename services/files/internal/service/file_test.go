package service

import (
	"bytes"
	"context"
	"errors"
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

// File operations

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

// File link operations

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

// Permissions

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

func (m *MockRepository) CheckFileAccess(ctx context.Context, fileLinkID, userID uuid.UUID) (model.FileAccessLevel, error) {
	args := m.Called(ctx, fileLinkID, userID)
	return args.Get(0).(model.FileAccessLevel), args.Error(1)
}

func (m *MockRepository) DeletePermissionsForUser(ctx context.Context, linkIDs []uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, linkIDs, userID)
	return args.Error(0)
}

// File groups

func (m *MockRepository) CreateFileGroup(ctx context.Context, group *model.FileGroup) error {
	args := m.Called(ctx, group)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	group.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetFileGroup(ctx context.Context, id uuid.UUID) (*model.FileGroup, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.FileGroup), args.Error(1)
}

func (m *MockRepository) DeleteFileGroup(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Group membership

func (m *MockRepository) AddUserToGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	args := m.Called(ctx, groupID, userID)
	return args.Error(0)
}

func (m *MockRepository) RemoveUserFromGroup(ctx context.Context, groupID, userID uuid.UUID) error {
	args := m.Called(ctx, groupID, userID)
	return args.Error(0)
}

func (m *MockRepository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// File-group links

func (m *MockRepository) AddFileLinkToGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error {
	args := m.Called(ctx, fileLinkID, groupID)
	return args.Error(0)
}

func (m *MockRepository) RemoveFileLinkFromGroup(ctx context.Context, fileLinkID, groupID uuid.UUID) error {
	args := m.Called(ctx, fileLinkID, groupID)
	return args.Error(0)
}

func (m *MockRepository) GetFileLinkGroups(ctx context.Context, fileLinkID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, fileLinkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockRepository) GetFilesByGroup(ctx context.Context, groupID uuid.UUID) ([]model.FileLink, error) {
	args := m.Called(ctx, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.FileLink), args.Error(1)
}

// Share links

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

// Message attachments

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

// Batch operations

func (m *MockRepository) GetFilesByLinkIDs(ctx context.Context, linkIDs []uuid.UUID) (map[uuid.UUID]*model.File, error) {
	args := m.Called(ctx, linkIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*model.File), args.Error(1)
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

// ========== Core File Operations Tests ==========

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
	// CheckFileAccess fails (function not available), falls back to individual permissions
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessNone, errors.New("function not found"))
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

func TestFileService_Download_WithGroupAccess(t *testing.T) {
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

	file := &model.File{
		ID:               fileID,
		Filename:         "abc123_test.txt",
		OriginalFilename: "test.txt",
		ContentType:      "text/plain",
		Size:             12,
		FilePath:         "2024/01/01/abc123_test.txt",
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	// CheckFileAccess succeeds with read access (user is in a group with read permission)
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessRead, nil)
	// GetFileLinkPermission is NOT called when CheckFileAccess succeeds
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
	// CheckFileAccess fails, falls back to individual permissions
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessNone, errors.New("function not found"))
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
	// CheckFileAccess fails, falls back to individual permissions
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessNone, errors.New("function not found"))
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)
	mockRepo.On("SoftDeleteFileLink", ctx, linkID).Return(nil)

	err := svc.Delete(ctx, linkID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_Delete_WithGroupAccess(t *testing.T) {
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

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	// CheckFileAccess succeeds with delete access (user is in a group with can_delete permission)
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessDelete, nil)
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
	// CheckFileAccess fails, falls back to individual permissions
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessNone, errors.New("function not found"))
	mockRepo.On("GetFileLinkPermission", ctx, linkID, userID).Return(permission, nil)

	err := svc.Delete(ctx, linkID, userID)

	assert.Error(t, err)
	assert.Equal(t, repository.ErrAccessDenied, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_Delete_NoGroupAccess(t *testing.T) {
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

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	// CheckFileAccess returns read access only (not delete)
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessRead, nil)

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
	// CheckFileAccess fails, falls back to individual permissions
	mockRepo.On("CheckFileAccess", ctx, linkID, userID).Return(model.FileAccessNone, errors.New("function not found"))
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

// ========== gRPC Methods Tests ==========

func TestFileService_CreateFileLink(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	fileID := uuid.New()

	file := &model.File{
		ID:               fileID,
		Filename:         "test.txt",
		OriginalFilename: "test.txt",
	}

	mockRepo.On("GetFile", ctx, fileID).Return(file, nil)
	mockRepo.On("CreateFileLink", ctx, mock.AnythingOfType("*model.FileLink")).Return(nil)
	mockRepo.On("CreateFileLinkPermission", ctx, mock.AnythingOfType("*model.FileLinkPermission")).Return(nil)

	linkID, err := svc.CreateFileLink(ctx, fileID, userID)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, linkID)

	mockRepo.AssertExpectations(t)
}

func TestFileService_RevokePermissions(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	linkIDs := []uuid.UUID{uuid.New(), uuid.New()}

	mockRepo.On("DeletePermissionsForUser", ctx, linkIDs, userID).Return(nil)

	err := svc.RevokePermissions(ctx, linkIDs, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_GetFileIDByLinkID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	fileID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:     linkID,
		FileID: fileID,
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)

	returnedFileID, err := svc.GetFileIDByLinkID(ctx, linkID)

	assert.NoError(t, err)
	assert.Equal(t, fileID, returnedFileID)
	mockRepo.AssertExpectations(t)
}

// ========== File Groups Tests ==========

func TestFileService_CreateFileGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	name := "test-moderate-group"

	mockRepo.On("CreateFileGroup", ctx, mock.AnythingOfType("*model.FileGroup")).Return(nil)

	group, err := svc.CreateFileGroup(ctx, name, true, true, false)

	assert.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, name, group.Name)
	assert.True(t, group.CanRead)
	assert.True(t, group.CanDelete)
	assert.False(t, group.CanTransfer)
	assert.NotEqual(t, uuid.Nil, group.ID)

	mockRepo.AssertExpectations(t)
}

func TestFileService_CreateFileGroup_ReadOnly(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	name := "test-read-group"

	mockRepo.On("CreateFileGroup", ctx, mock.AnythingOfType("*model.FileGroup")).Return(nil)

	group, err := svc.CreateFileGroup(ctx, name, true, false, false)

	assert.NoError(t, err)
	assert.NotNil(t, group)
	assert.Equal(t, name, group.Name)
	assert.True(t, group.CanRead)
	assert.False(t, group.CanDelete)
	assert.False(t, group.CanTransfer)

	mockRepo.AssertExpectations(t)
}

func TestFileService_DeleteFileGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	groupID := uuid.New()

	mockRepo.On("DeleteFileGroup", ctx, groupID).Return(nil)

	err := svc.DeleteFileGroup(ctx, groupID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_AddUserToGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	groupID := uuid.New()
	userID := uuid.New()

	mockRepo.On("AddUserToGroup", ctx, groupID, userID).Return(nil)

	err := svc.AddUserToGroup(ctx, groupID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_RemoveUserFromGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	groupID := uuid.New()
	userID := uuid.New()

	mockRepo.On("RemoveUserFromGroup", ctx, groupID, userID).Return(nil)

	err := svc.RemoveUserFromGroup(ctx, groupID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_AddFileLinkToGroups(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	fileLinkID := uuid.New()
	groupID1 := uuid.New()
	groupID2 := uuid.New()
	groupIDs := []uuid.UUID{groupID1, groupID2}

	mockRepo.On("AddFileLinkToGroup", ctx, fileLinkID, groupID1).Return(nil)
	mockRepo.On("AddFileLinkToGroup", ctx, fileLinkID, groupID2).Return(nil)

	err := svc.AddFileLinkToGroups(ctx, fileLinkID, groupIDs)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_AddFileLinkToGroups_EmptyGroups(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	fileLinkID := uuid.New()
	groupIDs := []uuid.UUID{}

	err := svc.AddFileLinkToGroups(ctx, fileLinkID, groupIDs)

	assert.NoError(t, err)
	// No repository calls expected
	mockRepo.AssertExpectations(t)
}

func TestFileService_GetFilesByGroup(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	groupID := uuid.New()
	linkID1 := uuid.New()
	linkID2 := uuid.New()

	expectedLinks := []model.FileLink{
		{ID: linkID1, FileID: uuid.New(), UploadedBy: uuid.New()},
		{ID: linkID2, FileID: uuid.New(), UploadedBy: uuid.New()},
	}

	mockRepo.On("GetFilesByGroup", ctx, groupID).Return(expectedLinks, nil)

	links, err := svc.GetFilesByGroup(ctx, groupID)

	assert.NoError(t, err)
	assert.Len(t, links, 2)
	assert.Equal(t, linkID1, links[0].ID)
	assert.Equal(t, linkID2, links[1].ID)

	mockRepo.AssertExpectations(t)
}

func TestFileService_RemoveUserFromAllGroupFiles(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	groupID1 := uuid.New()
	groupID2 := uuid.New()
	groupIDs := []uuid.UUID{groupID1, groupID2}

	linkID1 := uuid.New()
	linkID2 := uuid.New()
	linkID3 := uuid.New()

	// Files in group 1
	group1Links := []model.FileLink{
		{ID: linkID1, FileID: uuid.New()},
		{ID: linkID2, FileID: uuid.New()},
	}

	// Files in group 2
	group2Links := []model.FileLink{
		{ID: linkID3, FileID: uuid.New()},
	}

	mockRepo.On("GetFilesByGroup", ctx, groupID1).Return(group1Links, nil)
	mockRepo.On("RemoveUserFromGroup", ctx, groupID1, userID).Return(nil)
	mockRepo.On("GetFilesByGroup", ctx, groupID2).Return(group2Links, nil)
	mockRepo.On("RemoveUserFromGroup", ctx, groupID2, userID).Return(nil)
	// All link IDs from both groups
	mockRepo.On("DeletePermissionsForUser", ctx, mock.MatchedBy(func(linkIDs []uuid.UUID) bool {
		return len(linkIDs) == 3
	}), userID).Return(nil)

	err := svc.RemoveUserFromAllGroupFiles(ctx, groupIDs, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_RemoveUserFromAllGroupFiles_EmptyGroups(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	userID := uuid.New()
	groupIDs := []uuid.UUID{}

	err := svc.RemoveUserFromAllGroupFiles(ctx, groupIDs, userID)

	assert.NoError(t, err)
	// No repository calls expected
	mockRepo.AssertExpectations(t)
}

func TestFileService_GrantPermissions(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	uploaderID := uuid.New()
	linkID := uuid.New()
	userID1 := uuid.New()
	userID2 := uuid.New()

	fileLink := &model.FileLink{
		ID:         linkID,
		FileID:     uuid.New(),
		UploadedBy: uploaderID,
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	mockRepo.On("CreateFileLinkPermission", ctx, mock.AnythingOfType("*model.FileLinkPermission")).Return(nil).Times(2)

	err := svc.GrantPermissions(ctx, []uuid.UUID{linkID}, []uuid.UUID{userID1, userID2}, uploaderID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestFileService_GrantPermissions_NotUploader(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	mockStorage := new(MockStorage)
	svc := NewFileService(mockRepo, mockStorage, "http://localhost:8082")

	uploaderID := uuid.New()
	otherUserID := uuid.New()
	linkID := uuid.New()

	fileLink := &model.FileLink{
		ID:         linkID,
		FileID:     uuid.New(),
		UploadedBy: uploaderID, // Original uploader
	}

	mockRepo.On("GetFileLink", ctx, linkID).Return(fileLink, nil)
	// No CreateFileLinkPermission should be called because otherUserID is not the uploader

	err := svc.GrantPermissions(ctx, []uuid.UUID{linkID}, []uuid.UUID{uuid.New()}, otherUserID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
