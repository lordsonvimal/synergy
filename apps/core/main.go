package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/logger"
	"github.com/lordsonvimal/synergy/routes"
	"github.com/lordsonvimal/synergy/services/db"
)

func main() {
	ctx := context.Background()
	logger.InitLogger("Synergy App")
	log := logger.GetLogger()
	defer log.Flush()

	c, err := config.LoadConfig(ctx)
	if err != nil {
		log.Fatal(ctx, "Cannot load config", map[string]any{"error": err.Error()})
	}

	logger.ConfigureLogger(ctx, &logger.LoggerConfig{
		Environment: c.AppEnv,
		LogLevel:    c.LogLevel,
	})

	log.Info(ctx, fmt.Sprintf("Successfully loaded config using env %s with log level=%s", c.AppEnv, c.LogLevel), nil)

	db.InitPostgresDB(ctx)
	defer db.ClosePostgresDB(ctx)

	db.InitScyllaDB(ctx)
	defer db.CloseScyllaDB(ctx)

	pool := db.GetPostgresPool()
	scyllaSession := db.GetScyllaSession()

	db.InitDBs(pool, scyllaSession)

	router := setupRouter()

	// Server configuration with TLS
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", c.ServerPort),
		Handler: router,
	}

	// Graceful shutdown handling
	go func() {
		log.Info(ctx, fmt.Sprintf("Starting HTTPS server on port %s", c.ServerPort), nil)
		if err := srv.ListenAndServeTLS(c.ServerCert, c.ServerCertKey); err != nil && err != http.ErrServerClosed {
			log.Fatal(ctx, "Failed to start HTTPS server", map[string]any{"error": err.Error()})
		}
	}()

	gracefulShutdown(srv, log)
}

func setupRouter() *gin.Engine {
	router := gin.New()

	router.Use(gin.Recovery())

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup logger middleware
	router.Use(logger.LoggerMiddleware())

	// Inject the dbs - pgPool, scyllaSession using middleware
	router.Use(db.DBMiddleware())

	// Register routes
	routes.RegisterRoutes(router)

	return router
}

func gracefulShutdown(srv *http.Server, log logger.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	// Graceful shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info(ctx, "Shutting down server gracefully...", nil)

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(ctx, "Server shutdown failed", map[string]any{"error": err.Error()})
	}

	// Close DB connections
	log.Info(ctx, "Closing database connections...", nil)
	db.ClosePostgresDB(ctx)
	db.CloseScyllaDB(ctx)

	log.Info(ctx, "Server gracefully stopped", nil)
}
