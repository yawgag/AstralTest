package service

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/storage/cache"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockFileStorage struct{ mock.Mock }

func (m *mockFileStorage) SaveDoc(ctx context.Context, doc *entity.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}
func (m *mockFileStorage) DeleteDoc(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
func (m *mockFileStorage) SaveFile(id uuid.UUID, data *[]byte) error {
	args := m.Called(id, data)
	return args.Error(0)
}
func (m *mockFileStorage) RmFile(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *mockFileStorage) GetDoc(ctx context.Context, id uuid.UUID) (*entity.Document, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*entity.Document), args.Error(1)
}
func (m *mockFileStorage) GetFilePath(id uuid.UUID) string {
	args := m.Called(id)
	return args.String(0)
}
func (m *mockFileStorage) GetDocsList(ctx context.Context, ownerLogin, login, key, value string, limit int) ([]*entity.Document, error) {
	args := m.Called(ctx, ownerLogin, login, key, value, limit)
	return args.Get(0).([]*entity.Document), args.Error(1)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) SetOwner(login string, key cache.CacheKey, value cache.CachedDocResp) {
	m.Called(login, key, value)
}
func (m *mockCache) GetOwner(login string, key cache.CacheKey) (cache.CachedDocResp, bool) {
	args := m.Called(login, key)
	return args.Get(0).(cache.CachedDocResp), args.Bool(1)
}
func (m *mockCache) InvalidateOwnerList(login string) {
	m.Called(login)
}
func (m *mockCache) SetGrant(grantee, owner string, key cache.CacheKey, value cache.CachedDocResp) {
	m.Called(grantee, owner, key, value)
}
func (m *mockCache) GetGrant(grantee, owner string, key cache.CacheKey) (cache.CachedDocResp, bool) {
	args := m.Called(grantee, owner, key)
	return args.Get(0).(cache.CachedDocResp), args.Bool(1)
}
func (m *mockCache) InvalidateGrant(owner string, grantees []string) {
	m.Called(owner, grantees)
}

func TestHandleUploadingFile_Success(t *testing.T) {
	ctx := context.Background()
	token := uuid.New()
	fileData := []byte("test")
	doc := entity.Document{
		Token: token.String(),
		Name:  "doc.txt",
		File:  true,
		Grant: []string{"grantee1", "grantee2"},
	}

	session := new(mockSessionStorage)
	files := new(mockFileStorage)
	cache := new(mockCache)

	session.On("GetSession", ctx, token).Return("owner1", nil)
	files.On("SaveDoc", ctx, mock.AnythingOfType("*entity.Document")).Return(nil)
	files.On("SaveFile", mock.Anything, &fileData).Return(nil)
	cache.On("InvalidateOwnerList", "owner1").Return()
	cache.On("InvalidateGrant", "owner1", doc.Grant).Return()

	svc := NewWcsService(session, files, "tmp", cache)

	result, err := svc.HandleUploadingFile(ctx, doc, &fileData)
	require.NoError(t, err)
	assert.NotNil(t, result)

	session.AssertExpectations(t)
	files.AssertExpectations(t)
	cache.AssertExpectations(t)
}
