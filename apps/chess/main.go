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
	"github.com/lordsonvimal/synergy/apps/chess/config"
	"github.com/lordsonvimal/synergy/apps/chess/db"
	"github.com/lordsonvimal/synergy/apps/chess/db/sqlite"
	"github.com/lordsonvimal/synergy/apps/chess/logger"
	"github.com/lordsonvimal/synergy/apps/chess/server"
	"github.com/lordsonvimal/synergy/apps/chess/store"
	"github.com/lordsonvimal/synergy/apps/chess/ui/helpers"
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

	// Ensure DATA_DIR exists.
	dataDir := config.GetEnv("DATA_DIR", ".")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Fatal(ctx).Err(err).Str("DATA_DIR", dataDir).Msg("Cannot create DATA_DIR")
	}

	dbRepo, err := db.Open(config.DBPath(), sqlite.New)
	if err != nil {
		logger.Fatal(ctx).Err(err).Msg("Cannot open SQLite database")
	}
	defer dbRepo.Close()

	router := gin.New()

	gameStore := store.NewGameStore()

	router.Use(requestid.New())                                        // Add this for correlation IDs
	router.Use(logger.RedactedStructuredLogger(logger.GlobalLogger())) // Structured logging with token redaction (access_token, auth_token, etc.)
	router.Use(gin.Recovery())                                         // Use default recovery for panic logging/handling
	router.Use(store.StoreContext(gameStore))                          // Add gameStore to context
	router.Use(store.DBRepoContext(dbRepo))                            // Add db repository to context
	router.Use(server.CSRFMiddleware())

	// Asset pipeline: dist/ is produced by `yarn build` and contains
	// content-hashed files plus a manifest. Load the manifest once at startup
	// so helpers.Asset can map logical names to served paths.
	if err := helpers.LoadAssetManifest("./dist"); err != nil {
		logger.Fatal(ctx).Err(err).Msg("Cannot load asset manifest (run `yarn build`)")
	}
	router.GET("/static/*filepath", server.ServeStatic("./dist"))
	router.GET("/favicon.svg", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, helpers.Asset("favicon.svg"))
	})

	server.InitRoutes(router)

	srv := &http.Server{
		Addr:         ":3001",
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
