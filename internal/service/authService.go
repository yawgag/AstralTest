package service

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/storage"
	"AstralTest/pkg/appError"
	"context"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type auth struct {
	userStorage    storage.UserStorage
	sessionStorage storage.SessionStorage
	adminToken     uuid.UUID
}

type AuthService interface {
	Register(ctx context.Context, user *entity.User, token uuid.UUID) error
	Login(ctx context.Context, user *entity.User) (*uuid.UUID, error)
	Logout(ctx context.Context, token uuid.UUID) error
}

func NewAuthService(userStorage storage.UserStorage, sessionStorage storage.SessionStorage, adminToken uuid.UUID) AuthService {
	return &auth{
		userStorage:    userStorage,
		sessionStorage: sessionStorage,
		adminToken:     adminToken,
	}
}

// password validation function with rules:
// 1) the password is at least 8 characters long
// 2) the password has uppercase letters
// 3) the password has digit symbols
// 4) the password hass special symbols
func validatePassword(password string) error {
	if len(password) < 8 {
		return appError.BadRequest("invalid password (lenght below than 8 symbols)")
	}

	var (
		lowercase int
		uppercase int
		digits    int
		special   int
	)

	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			lowercase++
		case unicode.IsUpper(char):
			uppercase++
		case unicode.IsDigit(char):
			digits++
		default:
			special++
		}
	}

	// combine all errors in one string
	var passwordErrorsString string
	if lowercase == 0 {
		passwordErrorsString += "(lowercase characters are missing) "
	}
	if uppercase == 0 {
		passwordErrorsString += "(uppercase characters are missing) "
	}
	if digits == 0 {
		passwordErrorsString += "(digits are missing) "
	}
	if special == 0 {
		passwordErrorsString += "(special characters are missing) "
	}

	if len(passwordErrorsString) != 0 {
		return appError.BadRequest("invalid password: " + passwordErrorsString)
	}

	return nil
}

func (a *auth) Register(ctx context.Context, user *entity.User, token uuid.UUID) error {
	if err := validatePassword(user.Password); err != nil {
		return err
	}
	if len(user.Login) < 8 {
		return appError.BadRequest("login length can't be less than 8")
	}
	if token != a.adminToken {
		return appError.BadRequest("invalid admin token")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return appError.Internal()
	}
	user.PasswordHash = string(passwordHash)

	err = a.userStorage.AddUser(ctx, user)
	if err != nil {
		return err
	}

	return nil
}

func (a *auth) Login(ctx context.Context, user *entity.User) (*uuid.UUID, error) {
	originalUser, err := a.userStorage.GetUser(ctx, user.Login)
	if err != nil {
		return nil, err
	}

	// TODO: maybe change to http 400, not custom error
	validPassword := bcrypt.CompareHashAndPassword([]byte(originalUser.PasswordHash), []byte(user.Password))
	if validPassword != nil {
		return nil, appError.BadRequest("wrong password")
	}

	connToken, err := a.sessionStorage.CreateSession(ctx, user.Login)
	if err != nil {
		return nil, err
	}
	return connToken, nil
}

func (a *auth) Logout(ctx context.Context, token uuid.UUID) error {
	return a.sessionStorage.DeleteSession(ctx, token)
}
