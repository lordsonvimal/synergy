package db

import (
	"context"
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5"
)

type ScyllaRepository struct {
	session *gocql.Session
}

func NewScyllaRepository(session *gocql.Session) *ScyllaRepository {
	return &ScyllaRepository{session: session}
}

func (r *ScyllaRepository) Insert(ctx context.Context, query string, args ...any) error {
	return r.session.Query(query, args...).WithContext(ctx).Exec()
}

func (r *ScyllaRepository) QueryScylla(ctx context.Context, query string, args ...any) (*gocql.Iter, error) {
	return r.session.Query(query, args...).WithContext(ctx).Iter(), nil
}

func (r *ScyllaRepository) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return nil, errors.New("Query not supported by Scylla")
}

func (r *ScyllaRepository) GetRedis(ctx context.Context, key string) (string, error) {
	return "", errors.New("GetRedis not supported by Scylla")
}

func (r *ScyllaRepository) SetRedis(ctx context.Context, key string, value string, expiration time.Duration) error {
	return errors.New("SetRedis not supported by Scylla")
}

func (r *ScyllaRepository) Close() error {
	r.session.Close()
	return nil
}
