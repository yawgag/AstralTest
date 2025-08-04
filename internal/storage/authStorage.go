package storage

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/storage/postgres"
	"AstralTest/pkg/appError"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type auth struct {
	pool postgres.DBPool
}

type UserStorage interface {
	AddUser(ctx context.Context, user *entity.User) error
	GetUser(ctx context.Context, login string) (*entity.User, error)
}

func NewUserStorage(pool postgres.DBPool) UserStorage {
	return &auth{
		pool: pool,
	}
}

type SessionStorage interface {
	CreateSession(ctx context.Context, login string) (*uuid.UUID, error)
	GetSession(ctx context.Context, sessionId uuid.UUID) (string, error)
	DeleteSession(ctx context.Context, token uuid.UUID) error
}

func NewSessionStorage(pool postgres.DBPool) SessionStorage {
	return &auth{
		pool: pool,
	}
}

func (a *auth) AddUser(ctx context.Context, user *entity.User) error {
	query := `insert into users(login, password)
				values($1, $2)`

	_, err := a.pool.Exec(ctx, query, user.Login, user.PasswordHash)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgErr.Code == "23505" {
				if pgErr.ConstraintName == "unique_login" {
					return appError.BadRequest("unique login violation") // but maybe should return http 409 - conflict?
				}
			}
		}

		return appError.Internal()
	}

	return nil
}

func (a *auth) GetUser(ctx context.Context, login string) (*entity.User, error) {
	query := `select login, password
				from users
				where login = $1`
	var user entity.User
	err := a.pool.QueryRow(ctx, query, login).Scan(&user.Login, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appError.Unauthorized()
		}
		return nil, appError.Internal()
	}

	return &user, nil
}

func (a *auth) CreateSession(ctx context.Context, login string) (*uuid.UUID, error) {
	query := `insert into sessions(login)
				values($1)
				returning session_id`
	var connToken uuid.UUID
	err := a.pool.QueryRow(ctx, query, login).Scan(&connToken)
	if err != nil {
		return nil, appError.Internal()
	}

	return &connToken, nil
}

func (a *auth) GetSession(ctx context.Context, sessionId uuid.UUID) (string, error) {
	query := `select login
				from sessions
				where session_id = $1`

	var login string

	err := a.pool.QueryRow(ctx, query, sessionId).Scan(&login)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", appError.Unauthorized()
		}
		return "", appError.Internal()
	}
	return login, nil
}

func (a *auth) DeleteSession(ctx context.Context, token uuid.UUID) error {
	query := `delete from sessions
				where session_id = $1`
	_, err := a.pool.Exec(ctx, query, token)
	if err != nil {
		return appError.Internal()
	}

	return nil
}
