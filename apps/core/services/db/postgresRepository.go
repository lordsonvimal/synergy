package db

import (
	"context"
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Insert(ctx context.Context, query string, args ...any) error {
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *PostgresRepository) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return r.pool.Query(ctx, query, args...)
}

func (r *PostgresRepository) QueryScylla(ctx context.Context, query string, args ...any) (*gocql.Iter, error) {
	return nil, errors.New("QueryScylla not supported by PostgreSQL")
}

func (r *PostgresRepository) GetRedis(ctx context.Context, key string) (string, error) {
	return "", errors.New("GetRedis not supported by PostgreSQL")
}

func (r *PostgresRepository) SetRedis(ctx context.Context, key string, value string, expiration time.Duration) error {
	return errors.New("SetRedis not supported by PostgreSQL")
}

func (r *PostgresRepository) Close() error {
	r.pool.Close()
	return nil
}
