package db

import (
	"context"
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) GetRedis(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisRepository) SetRedis(ctx context.Context, key string, value string, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisRepository) Insert(ctx context.Context, query string, args ...interface{}) error {
	return errors.New("Insert not supported by Redis")
}

func (r *RedisRepository) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("Query not supported by Redis")
}

func (r *RedisRepository) QueryScylla(ctx context.Context, query string, args ...interface{}) (*gocql.Iter, error) {
	return nil, errors.New("QueryScylla not supported by Redis")
}

func (r *RedisRepository) Close() error {
	return r.client.Close()
}
