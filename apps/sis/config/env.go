package config

import (
	"context"
	"os"

	"github.com/joho/godotenv"
	"github.com/lordsonvimal/synergy/apps/sis/logger"
)

// LoadEnv loads environment variables from a .env file.
// It searches for the file in the current working directory.
// This function should be called once at the start of main.go.
func LoadEnv(ctx context.Context) {
	// Attempt to load .env file.
	// godotenv.Load() automatically searches for ".env" in the current directory.
	// If the file is not found, or if there's an error, we print a warning but don't panic,
	// allowing the application to proceed using system environment variables and fallbacks.
	err := godotenv.Load()

	if err != nil {
		// Only log a critical error if the file was specified but failed to load,
		// otherwise, it's normal to proceed without a local .env file (e.g., in Docker/Kubernetes).
		logger.Warn(ctx).Err(err).Msgf("⚠️ WARNING: Could not load .env file. Falling back to system environment variables. Error: %v\n", err)
	} else {
		logger.Info(ctx).Msg("✅ Successfully loaded environment variables from .env")
	}
}

// Example Helper: GetEnv provides a safe way to retrieve an environment variable
// with a fallback value.
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
