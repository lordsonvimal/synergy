package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/apps/sis/config"
	"github.com/lordsonvimal/synergy/apps/sis/db"
	"github.com/lordsonvimal/synergy/apps/sis/logger"
	"github.com/lordsonvimal/synergy/apps/sis/server"
	"github.com/lordsonvimal/synergy/apps/sis/shared/appctx"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()

	// Attach logger to context
	ctx = log.Logger.WithContext(ctx)

	config.LoadEnv(ctx)

	mode := config.GetEnv("GIN_MODE", "")
	isProduction := mode == "release"
	logger.InitLogger(isProduction)

	logger.Info(ctx).Str("GIN_MODE", mode).Bool("isProduction", isProduction).Msg("Checking Production mode")
	if isProduction {
		gin.SetMode(gin.ReleaseMode)
	}

	database, err := db.InitSQLiteDB()
	if err != nil {
		logger.Fatal(ctx).Err(err).Msg("Failed to initialize SQLite database")
	}
	defer database.Close()

	router := gin.New()

	router.Use(requestid.New())                                        // Add this for correlation IDs
	router.Use(logger.RedactedStructuredLogger(logger.GlobalLogger())) // Structured logging with token redaction (access_token, auth_token, etc.)
	router.Use(gin.Recovery())                                         // Use default recovery for panic logging/handling
	router.Use(appctx.Middleware(database))

	router.Static("/static", "./dist")
	router.StaticFile("/favicon.ico", "assets/favicon.ico")

	server.InitRoutes(router)

	srv := &http.Server{
		Addr:         ":3002",
		Handler:      router,
		ReadTimeout:  1 * time.Minute,
		WriteTimeout: 1 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(ctx).Err(err).Msg("Server listen error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx).Msg("Shutdown signal received. Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal(ctx).Err(err).Msg("Server forced to shutdown")
	}

	logger.Info(ctx).Msg("Server exiting gracefully.")
}
