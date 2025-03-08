package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/lordsonvimal/synergy/config"
	"github.com/lordsonvimal/synergy/logger"
	"github.com/lordsonvimal/synergy/routes"
	"github.com/lordsonvimal/synergy/services/db"
)

func main() {
	r := gin.Default()

	logger.InitLogger("Synergy App")
	log := logger.GetLogger()

	c, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Cannot load config", map[string]interface{}{"error": err.Error()})
	}

	err = db.ValidateSchema()
	if err != nil {
		log.Fatal(fmt.Sprintf("❌ Schema validation failed: %v", err), nil)
		os.Exit(1)
	}

	err = db.RunMigrations()
	if err != nil {
		log.Fatal(fmt.Sprintf("❌ Migration error: %v", err), nil)
		os.Exit(1)
	}

	db.InitPostgresDB()
	defer db.ClosePostgresDB()

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		logger.GetLogger().Info("Shutting down server gracefully", nil)
		db.ClosePostgresDB()
		os.Exit(0)
	}()

	httpsCert := c.ServerCert
	httpsKey := c.ServerCertKey

	routes.RegisterRoutes(r)

	log.Info(fmt.Sprintf("Starting HTTPS server on port %s", c.ServerPort), nil)
	// Start the server with HTTPS
	err = r.RunTLS(c.ServerPort, httpsCert, httpsKey)
	if err != nil {
		log.Fatal("Failed to start HTTPS server", map[string]interface{}{"error": err.Error()})
	}
}
