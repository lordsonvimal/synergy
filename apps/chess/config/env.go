package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
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

// Make sure application is loaded with essential envs
func ValidateRequiredEnv(ctx context.Context) {
	required := []string{"WOPI_CLIENT_TOKEN_SECRET", "WOPI_APP_TOKEN_SECRET", "WOPI_HOST_URL", "WOPI_HOSTPAGE_TOKEN_SECRET", "WOPI_HOSTPAGE_TOKEN_ISSUER"}
	for _, key := range required {
		if GetEnv(key, "") == "" {
			msg := fmt.Sprintf("FATAL: Required environment variable %s is not set", key)
			logger.Fatal(ctx).Msg(msg)
		}
	}

	// Validate HTTPS
	if !strings.HasPrefix(os.Getenv("WOPI_HOST_URL"), "https://") {
		logger.Fatal(ctx).Msg("FATAL: WOPI_HOST_URL must use HTTPS protocol")
	}
}
