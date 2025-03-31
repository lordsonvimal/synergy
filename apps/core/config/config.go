package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/lordsonvimal/synergy/logger"
)

// Config holds all configuration values with validation tags
type Config struct {
	AppEnv                  string   `validate:"required"`
	LogLevel                string   `validate:"required"`
	GoogleOauthClientId     string   `validate:"required"`
	GoogleOauthClientSecret string   `validate:"required"`
	GoogleOauthRedirectUrl  string   `validate:"required"`
	ServerCert              string   `validate:"required"`
	ServerCertKey           string   `validate:"required"`
	ServerPort              string   `validate:"required,numeric"`
	PostgresURL             string   `validate:"required,url"`
	PostgresConnMaxLifetime int32    `validate:"required,numeric"`
	PostgresConnMaxIdleTime int32    `validate:"required,numeric"`
	PostgresMinConns        int32    `validate:"required,numeric"`
	PostgresMaxConns        int32    `validate:"required,numeric"`
	RedisURL                string   `validate:"required,url"`
	JWTSecret               string   `validate:"required"`
	ScyllaHosts             []string `validate:"required"`
	ScyllaKeyspace          string   `validate:"required"`
	ScyllaTimeout           int32    `validate:"required"`
	ScyllaConnectTimeout    int32    `validate:"required"`
	ScyllaMaxConns          int32    `validate:"required"`
}

// defaultConfig defines fallback values if environment variables are missing
var defaultConfig = Config{
	AppEnv:                  "development",
	LogLevel:                "debug",
	GoogleOauthClientId:     "",
	GoogleOauthClientSecret: "",
	GoogleOauthRedirectUrl:  "http://localhost:3001/callback",
	ServerCert:              "",
	ServerCertKey:           "",
	ServerPort:              "8080",
	PostgresURL:             "postgres://postgres:postgres@localhost:5433/synergy",
	PostgresConnMaxLifetime: 500,
	PostgresConnMaxIdleTime: 60,
	PostgresMinConns:        1,
	PostgresMaxConns:        5,
	RedisURL:                "redis://localhost:6379",
	JWTSecret:               "supersecret",
	ScyllaHosts:             []string{"localhost:9042"},
	ScyllaKeyspace:          "synergy",
	ScyllaTimeout:           10,
	ScyllaConnectTimeout:    5,
	ScyllaMaxConns:          5,
}

var instance *Config

// GetConfig returns the config instance
func GetConfig() *Config {
	return instance
}

// LoadConfig loads and validates environment variables
func LoadConfig(ctx context.Context) (*Config, error) {
	if instance != nil {
		return instance, nil
	}

	log := logger.GetLogger()
	log.Info(ctx, "Loading configuration...", nil)

	if err := godotenv.Load(); err != nil {
		log.Warn(ctx, "No .env file found, using system environment variables", nil)
	}

	config := &Config{
		AppEnv:                  getEnv(ctx, "APP_ENV", defaultConfig.AppEnv),
		LogLevel:                getEnv(ctx, "LOG_LEVEL", defaultConfig.LogLevel),
		GoogleOauthClientId:     getEnv(ctx, "GOOGLE_OAUTH_CLIENT_ID", defaultConfig.GoogleOauthClientId),
		GoogleOauthClientSecret: getEnv(ctx, "GOOGLE_OAUTH_CLIENT_SECRET", defaultConfig.GoogleOauthClientSecret),
		GoogleOauthRedirectUrl:  getEnv(ctx, "GOOGLE_OAUTH_REDIRECT_URL", defaultConfig.GoogleOauthRedirectUrl),
		ServerCert:              getEnv(ctx, "HTTPS_SERVER_CERT", defaultConfig.ServerCert),
		ServerCertKey:           getEnv(ctx, "HTTPS_SERVER_KEY", defaultConfig.ServerCertKey),
		ServerPort:              getEnv(ctx, "SERVER_PORT", defaultConfig.ServerPort),
		PostgresURL:             getEnv(ctx, "POSTGRES_URL", defaultConfig.PostgresURL),
		PostgresConnMaxLifetime: getEnvInt("POSTGRES_CONN_MAX_LIFETIME", defaultConfig.PostgresConnMaxLifetime),
		PostgresConnMaxIdleTime: getEnvInt("POSTGRES_CONN_MAX_IDLE_TIME", defaultConfig.PostgresConnMaxIdleTime),
		PostgresMinConns:        getEnvInt("POSTGRES_MIN_CONN", defaultConfig.PostgresMinConns),
		PostgresMaxConns:        getEnvInt("POSTGRES_MAX_CONN", defaultConfig.PostgresMaxConns),
		RedisURL:                getEnv(ctx, "REDIS_URL", defaultConfig.RedisURL),
		JWTSecret:               getEnv(ctx, "JWT_SECRET", defaultConfig.JWTSecret),
		ScyllaHosts:             getEnvList("SCYLLA_HOSTS", defaultConfig.ScyllaHosts),
		ScyllaKeyspace:          getEnv(ctx, "SCYLLA_KEYSPACE", defaultConfig.ScyllaKeyspace),
		ScyllaTimeout:           getEnvInt("SCYLLA_TIMEOUT", defaultConfig.ScyllaTimeout),
		ScyllaConnectTimeout:    getEnvInt("SCYLLA_CONNECT_TIMEOUT", defaultConfig.ScyllaConnectTimeout),
		ScyllaMaxConns:          getEnvInt("SCYLLA_MAX_CONNS", defaultConfig.ScyllaMaxConns),
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		log.Info(ctx, "Has error", nil)
		var validationErrors validator.ValidationErrors

		if errors.As(err, &validationErrors) { // Safe type assertion
			for _, e := range validationErrors {
				msg := fmt.Sprintf("Invalid field %s", e.Field())
				log.Error(ctx, msg, map[string]any{
					"field": e.Field(),
					"error": e.Tag(),
				})
			}
		} else {
			log.Error(ctx, "Unexpected validation error", map[string]any{
				"error": err.Error(),
			})
		}

		return nil, err // Ensure caller properly handles the error
	}

	log.Info(ctx, "Configuration loaded successfully", map[string]any{})

	instance = config

	return config, nil
}

// getEnv fetches an environment variable or returns a default value
func getEnv(ctx context.Context, key, defaultVal string) string {
	log := logger.GetLogger()
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Warn(ctx, "Missing environment variable, using default", map[string]any{
		"key":     key,
		"default": defaultVal,
	})
	return defaultVal
}

func getEnvInt(key string, defaultValue int32) int32 {
	if value, exists := os.LookupEnv(key); exists {
		var intValue int32
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvList(key string, defaultValue []string) []string {
	return defaultValue
}
