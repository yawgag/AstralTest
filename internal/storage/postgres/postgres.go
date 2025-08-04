package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBPool interface {
	Close()
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PgxDBPool struct {
	pool *pgxpool.Pool
}

func (p *PgxDBPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *PgxDBPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

func (p *PgxDBPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.pool.Exec(ctx, sql, args...)
}

func (p *PgxDBPool) Close() {
	p.pool.Close()
}

func InitDb(dbAddr string) (DBPool, error) {
	dbConfig, err := pgxpool.ParseConfig(dbAddr)
	if err != nil {
		return nil, err
	}

	dbConfig.MaxConns = 20
	dbConfig.MinConns = 1
	dbConfig.MaxConnLifetime = time.Hour

	DbConnPool, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		return nil, err
	}

	err = DbConnPool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return &PgxDBPool{pool: DbConnPool}, nil
}
