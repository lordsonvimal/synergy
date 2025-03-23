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
	logger.InitLogger("Synergy App")
	log := logger.GetLogger()

	c, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Cannot load config", map[string]any{"error": err.Error()})
	}

	// err = db.ValidateSchema()
	// if err != nil {
	// 	log.Fatal(fmt.Sprintf("❌ Schema validation failed: %v", err), nil)
	// 	os.Exit(1)
	// }

	// err = db.RunMigrations()
	// if err != nil {
	// 	log.Fatal(fmt.Sprintf("❌ Migration error: %v", err), nil)
	// 	os.Exit(1)
	// }

	db.InitPostgresDB()
	defer db.ClosePostgresDB()

	db.InitScyllaDB()
	defer db.CloseScyllaDB()

	// Create repositories
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
		log.Info(fmt.Sprintf("Starting HTTPS server on port %s", c.ServerPort), nil)
		if err := srv.ListenAndServeTLS(c.ServerCert, c.ServerCertKey); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start HTTPS server", map[string]any{"error": err.Error()})
		}
	}()

	gracefulShutdown(srv, log)
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001", "https://localhost:3001"}, // Allowed origins
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"}, // Allow all headers
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowWildcard:    true,
		AllowOriginFunc: func(origin string) bool {
			return true // Allow all origins
		},
		MaxAge: 12 * time.Hour, // Cache preflight requests
	}))

	// Inject the container middleware
	router.Use(db.DBMiddleware())

	// Register routes
	routes.RegisterRoutes(router)

	return router
}

func gracefulShutdown(srv *http.Server, log logger.Logger) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Info("Shutting down server gracefully...", nil)

	// Graceful shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server shutdown failed", map[string]interface{}{"error": err.Error()})
	}

	// Close DB connections
	log.Info("Closing database connections...", nil)
	db.ClosePostgresDB()
	db.CloseScyllaDB()

	log.Info("Server gracefully stopped", nil)
}
