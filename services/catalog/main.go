package main

import (
	"catalog/internal/config"
	"catalog/internal/database"
	"catalog/internal/httpapi"
	"catalog/internal/identity"
	"catalog/internal/storage"
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

	// Connect to Identity Service
	identityClient, err := identity.NewClient(cfg.IdentityBaseURL)
	if err != nil {
		slog.Error("identity setup failed", "reason", err.Error())
		os.Exit(1)
	}

	// Initialize Cloudflare R2
	var imageStore httpapi.ImageStore
	if cfg.R2Endpoint != "" || cfg.R2Bucket != "" || cfg.R2AccessKeyID != "" || cfg.R2SecretAccessKey != "" {
		r2, err := storage.NewR2(cfg.R2Endpoint, cfg.R2Bucket, cfg.R2AccessKeyID, cfg.R2SecretAccessKey)
		if err != nil {
			slog.Error("image storage setup failed", "reason", err.Error())
			os.Exit(1)
		}
		imageStore = r2
	}

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
	router.POST(httpapi.ImageUploadPath, httpapi.RequireAdmin(identityClient), httpapi.UploadImage(imageStore))

	// Start server
	address := fmt.Sprintf(":%d", cfg.AppPort)
	if err := router.Run(address); err != nil {
		slog.Error("server failed to start", "reason", err.Error())
		os.Exit(1)
	}
}
