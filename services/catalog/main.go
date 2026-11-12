package main

import (
	"catalog/internal/config"
	"catalog/internal/database"
	"catalog/internal/handler"
	"catalog/internal/identity"
	"catalog/internal/repository"
	"catalog/internal/router"
	"catalog/internal/service"
	"catalog/internal/storage"
	"fmt"
	"log/slog"
	"os"
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
	var imageStore handler.ImageStore
	var imageChecker service.ImageChecker
	if cfg.R2Endpoint != "" || cfg.R2Bucket != "" || cfg.R2AccessKeyID != "" || cfg.R2SecretAccessKey != "" {
		r2, err := storage.NewR2(cfg.R2Endpoint, cfg.R2Bucket, cfg.R2AccessKeyID, cfg.R2SecretAccessKey)
		if err != nil {
			slog.Error("image storage setup failed", "reason", err.Error())
			os.Exit(1)
		}
		imageStore = r2
		imageChecker = r2
	}

	// Initialize Repository
	brandRepository := repository.NewBrandRepository(db)

	// Initialize Service
	brandService := service.NewBrandService(brandRepository, imageChecker)

	// Initialize Handler
	brandHandler := handler.NewBrandHandler(brandService)

	// Initialize Gin
	engine, err := router.New(identityClient, brandHandler, imageStore)
	if err != nil {
		slog.Error("HTTP setup failed", "reason", err.Error())
		os.Exit(1)
	}

	// Start server
	address := fmt.Sprintf(":%d", cfg.AppPort)
	if err := engine.Run(address); err != nil {
		slog.Error("server failed to start", "reason", err.Error())
		os.Exit(1)
	}
}
