package db

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/logger"
)

var (
	dbPool *pgxpool.Pool
	once   sync.Once
)

func InitPostgresDB() {
	once.Do(func() {
		log := logger.GetLogger()
		c, err := config.LoadConfig()

		if err != nil {
			log.Fatal("Failed to load config", map[string]interface{}{"error": err.Error()})
		}

		poolConfig, err := pgxpool.ParseConfig(c.PostgresURL)

		if err != nil {
			log.Fatal("Failed to parse DB config", map[string]interface{}{"error": err.Error()})
		}

		// Set connection pool settings
		poolConfig.MaxConns = c.PostgresMaxConns
		poolConfig.MinConns = c.PostgresMinConns
		poolConfig.MaxConnLifetime = time.Duration(c.PostgresConnMaxLifetime) * time.Second
		poolConfig.MaxConnIdleTime = time.Duration(c.PostgresConnMaxIdleTime) * time.Second

		// Create connection pool
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Add timeout context
		defer cancel()

		// Create connection pool
		dbPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			log.Fatal("Failed to create DB pool", map[string]any{"error": err.Error()})
		}

		log.Info("Connected to PostgreSQL", nil)
	})
}

// GetPostgresPool returns the connection pool instance
func GetPostgresPool() *pgxpool.Pool {
	if dbPool == nil {
		log.Fatal("Database not initialized. Call InitPostgresDB() first.")
	}
	return dbPool
}

func GetPgxPoolFromCtx(ctx context.Context) (*pgxpool.Pool, error) {
	db, ok := ctx.Value(DBKey).(*DBs)
	if !ok {
		return nil, errors.New("pool not found in context. set it in middleware")
	}
	return db.PgPool, nil
}

func Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool := GetPostgresPool()

	err := pool.Ping(ctx)

	if err != nil {
		return err
	}

	return nil
}

// closes the connection pool
func ClosePostgresDB() {
	log := logger.GetLogger()
	if dbPool != nil {
		dbPool.Close()
		log.Info("Database connection closed", nil)
	}
}
