package service

import (
	"AstralTest/internal/models/entity"
	"AstralTest/pkg/appError"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type mockUserStorage struct{ mock.Mock }

func (m *mockUserStorage) AddUser(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserStorage) GetUser(ctx context.Context, login string) (*entity.User, error) {
	args := m.Called(ctx, login)
	if user, ok := args.Get(0).(*entity.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

type mockSessionStorage struct{ mock.Mock }

func (m *mockSessionStorage) CreateSession(ctx context.Context, login string) (*uuid.UUID, error) {
	args := m.Called(ctx, login)
	if id, ok := args.Get(0).(*uuid.UUID); ok {
		return id, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockSessionStorage) GetSession(ctx context.Context, sessionId uuid.UUID) (string, error) {
	args := m.Called(ctx, sessionId)
	return args.String(0), args.Error(1)
}

func (m *mockSessionStorage) DeleteSession(ctx context.Context, token uuid.UUID) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func TestAuthService_Login(t *testing.T) {
	adminToken := uuid.New()
	correctPassword := "StrongPass1!"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

	testCases := []struct {
		name                string
		loginInput          *entity.User
		storedUser          *entity.User
		getUserErr          error
		createSessionID     *uuid.UUID
		createSessErr       error
		expectErr           bool
		errCode             int
		expectCreateSession bool
	}{
		{
			name: "successful login",
			loginInput: &entity.User{
				Login:    "testuser",
				Password: correctPassword,
			},
			storedUser: &entity.User{
				Login:        "testuser",
				PasswordHash: string(hashedPassword),
			},
			createSessionID:     func() *uuid.UUID { id := uuid.New(); return &id }(),
			expectErr:           false,
			expectCreateSession: true,
		},
		{
			name: "wrong password",
			loginInput: &entity.User{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			storedUser: &entity.User{
				Login:        "testuser",
				PasswordHash: string(hashedPassword),
			},
			getUserErr:          appError.BadRequest("test"),
			expectErr:           true,
			errCode:             400,
			expectCreateSession: false,
		},
		{
			name: "user not found",
			loginInput: &entity.User{
				Login:    "unknown",
				Password: "doesntmatter",
			},
			getUserErr:          appError.Unauthorized(),
			expectErr:           true,
			errCode:             401,
			expectCreateSession: false,
		},
		{
			name: "error creating session",
			loginInput: &entity.User{
				Login:    "testuser",
				Password: correctPassword,
			},
			storedUser: &entity.User{
				Login:        "testuser",
				PasswordHash: string(hashedPassword),
			},
			createSessErr:       appError.Internal(),
			expectErr:           true,
			errCode:             500,
			expectCreateSession: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userMock := new(mockUserStorage)
			sessMock := new(mockSessionStorage)
			auth := NewAuthService(userMock, sessMock, adminToken)
			ctx := context.Background()

			if tc.getUserErr != nil {
				userMock.On("GetUser", ctx, tc.loginInput.Login).Return(nil, tc.getUserErr).Once()
			} else {
				userMock.On("GetUser", ctx, tc.loginInput.Login).Return(tc.storedUser, nil).Once()
			}

			if tc.expectCreateSession {
				sessMock.On("CreateSession", ctx, tc.loginInput.Login).Return(tc.createSessionID, tc.createSessErr).Once()
			}

			token, err := auth.Login(ctx, tc.loginInput)
			if tc.expectErr {
				require.Error(t, err)
				appErr, ok := err.(appError.AppError)
				assert.True(t, ok)
				assert.Equal(t, tc.errCode, appErr.Code())
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				require.NotNil(t, token)
				assert.Equal(t, *tc.createSessionID, *token)
			}

			userMock.AssertExpectations(t)
			sessMock.AssertExpectations(t)
		})
	}
}

func TestAuthService_Register(t *testing.T) {
	adminToken := uuid.New()

	testCases := []struct {
		name        string
		user        *entity.User
		inputToken  uuid.UUID
		expectError bool
		errorCode   int
		mockAddErr  error
	}{
		{
			name: "successful registration",
			user: &entity.User{
				Login:    "ValidLogin",
				Password: "GoodPass1!",
			},
			inputToken:  adminToken,
			expectError: false,
		},
		{
			name: "too short login",
			user: &entity.User{
				Login:    "short",
				Password: "GoodPass1!",
			},
			inputToken:  adminToken,
			expectError: true,
			errorCode:   400,
		},
		{
			name: "invalid password (no digit)",
			user: &entity.User{
				Login:    "ValidLogin",
				Password: "NoDigits!",
			},
			inputToken:  adminToken,
			expectError: true,
			errorCode:   400,
		},
		{
			name: "invalid admin token",
			user: &entity.User{
				Login:    "ValidLogin",
				Password: "GoodPass1!",
			},
			inputToken:  uuid.New(),
			expectError: true,
			errorCode:   400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockUser := new(mockUserStorage)
			mockSess := new(mockSessionStorage)
			auth := NewAuthService(mockUser, mockSess, adminToken)
			ctx := context.Background()

			if !tc.expectError || tc.mockAddErr != nil {
				mockUser.On("AddUser", ctx, mock.AnythingOfType("*entity.User")).Return(tc.mockAddErr).Maybe()
			}

			err := auth.Register(ctx, tc.user, tc.inputToken)
			if tc.expectError {
				require.Error(t, err)
				appErr, ok := err.(appError.AppError)
				assert.True(t, ok)
				assert.Equal(t, tc.errorCode, appErr.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	adminToken := uuid.New()
	sessionID := uuid.New()

	mockUser := new(mockUserStorage)
	mockSess := new(mockSessionStorage)
	auth := NewAuthService(mockUser, mockSess, adminToken)
	ctx := context.Background()

	mockSess.On("DeleteSession", ctx, sessionID).Return(nil).Once()
	err := auth.Logout(ctx, sessionID)
	require.NoError(t, err)
	mockSess.AssertExpectations(t)
}
