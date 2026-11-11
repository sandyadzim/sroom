package main

import (
	"catalog/internal/config"
	"catalog/internal/database"
	"catalog/internal/httpapi"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "reason", err.Error())
		os.Exit(1)
	}

	slog.Info("configuration loaded", "port", cfg.AppPort)

	// Connect to PostgreSQL
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "reason", err.Error())
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("database pool unavailable")
		os.Exit(1)
	}
	defer sqlDB.Close()

	slog.Info("database connected")

	// Initialize Gin
	router := gin.New()
	if err := httpapi.Setup(router); err != nil {
		slog.Error("HTTP setup failed", "reason", err.Error())
		os.Exit(1)
	}

	// Route
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Start server
	address := fmt.Sprintf(":%d", cfg.AppPort)
	if err := router.Run(address); err != nil {
		slog.Error("server failed to start", "reason", err.Error())
		os.Exit(1)
	}
}
