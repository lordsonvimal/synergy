package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/lordsonvimal/synergy/logger"
)

// Config holds all configuration values with validation tags
type Config struct {
	ServerCert              string `validate:"required"`
	ServerCertKey           string `validate:"required"`
	ServerPort              string `validate:"required,numeric"`
	PostgresURL             string `validate:"required,url"`
	PostgresConnMaxLifetime int32  `validate:"required,numeric"`
	PostgresConnMaxIdleTime int32  `validate:"required,numeric"`
	PostgresMinConns        int32  `validate:"required,numeric"`
	PostgresMaxConns        int32  `validate:"required,numeric"`
	RedisURL                string `validate:"required,url"`
	JWTSecret               string `validate:"required"`
}

// defaultConfig defines fallback values if environment variables are missing
var defaultConfig = Config{
	ServerCert:              "",
	ServerCertKey:           "",
	ServerPort:              "8080",
	PostgresURL:             "postgres://postgres:postgres@localhost:5432/synergy",
	PostgresConnMaxLifetime: 500,
	PostgresConnMaxIdleTime: 60,
	PostgresMinConns:        1,
	PostgresMaxConns:        5,
	RedisURL:                "redis://localhost:6379",
	JWTSecret:               "supersecret",
}

// LoadConfig loads and validates environment variables
func LoadConfig() (*Config, error) {
	log := logger.GetLogger()
	log.Info("Loading configuration...", nil)

	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, using system environment variables", nil)
	}

	config := &Config{
		ServerCert:              getEnv("HTTPS_SERVER_CERT", defaultConfig.ServerCert),
		ServerCertKey:           getEnv("HTTPS_SERVER_KEY", defaultConfig.ServerCertKey),
		ServerPort:              getEnv("SERVER_PORT", defaultConfig.ServerPort),
		PostgresURL:             getEnv("POSTGRES_URL", defaultConfig.PostgresURL),
		PostgresConnMaxLifetime: getEnvInt("POSTGRES_CONN_MAX_LIFETIME", defaultConfig.PostgresConnMaxLifetime),
		PostgresConnMaxIdleTime: getEnvInt("POSTGRES_CONN_MAX_IDLE_TIME", defaultConfig.PostgresConnMaxIdleTime),
		PostgresMinConns:        getEnvInt("POSTGRES_MIN_CONN", defaultConfig.PostgresMinConns),
		PostgresMaxConns:        getEnvInt("POSTGRES_MAX_CONN", defaultConfig.PostgresMaxConns),
		RedisURL:                getEnv("REDIS_URL", defaultConfig.RedisURL),
		JWTSecret:               getEnv("JWT_SECRET", defaultConfig.JWTSecret),
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			log.Error("Invalid configuration field", map[string]interface{}{
				"field": e.Field(),
				"error": e.Tag(),
			})
		}
		return nil, err
	}

	log.Info("Configuration loaded successfully", map[string]interface{}{
		"ServerPort":  config.ServerPort,
		"PostgresURL": config.PostgresURL,
		"RedisURL":    config.RedisURL,
	})

	return config, nil
}

// getEnv fetches an environment variable or returns a default value
func getEnv(key, defaultVal string) string {
	log := logger.GetLogger()
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Warn("Missing environment variable, using default", map[string]interface{}{
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
