package db

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5"
)

// Common repository interface for PostgreSQL, Scylla, and Redis
type Repository interface {
	Insert(ctx context.Context, query string, args ...interface{}) error
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
	QueryScylla(ctx context.Context, query string, args ...interface{}) (*gocql.Iter, error)
	GetRedis(ctx context.Context, key string) (string, error)
	SetRedis(ctx context.Context, key string, value string, expiration time.Duration) error
	Close() error
}
