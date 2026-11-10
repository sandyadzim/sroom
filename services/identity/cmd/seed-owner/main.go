package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service", "identity")
	if err := run(logger); err != nil {
		logger.Error("owner seed failed", "operation", "seed_owner", "reason", "seed_failed")
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	ownerEmail := strings.ToLower(strings.TrimSpace(os.Getenv("OWNER_EMAIL")))
	ownerFullName := strings.TrimSpace(os.Getenv("OWNER_FULL_NAME"))
	ownerPassword := os.Getenv("OWNER_PASSWORD")

	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	if ownerEmail == "" {
		return errors.New("OWNER_EMAIL is required")
	}

	if ownerFullName == "" {
		return errors.New("OWNER_FULL_NAME is required")
	}

	if err := dto.ValidateRegister(dto.RegisterRequest{
		FullName: ownerFullName,
		Email:    ownerEmail,
		Password: ownerPassword,
	}); err != nil {
		return fmt.Errorf("validate OWNER: %w", err)
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		TranslateError: true,
		Logger:         gormlogger.Discard,
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database connection: %w", err)
	}
	defer sqlDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	created := false

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(?)",
			int64(16052026),
		).Error; err != nil {
			return fmt.Errorf("acquire seeder lock: %w", err)
		}

		var existingOwner model.User

		err := tx.
			Where("role = ?", model.RoleOwner).
			First(&existingOwner).Error

		if err == nil {
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check existing OWNER: %w", err)
		}

		// Jangan mengubah CUSTOMER/ADMIN menjadi OWNER
		// hanya karena emailnya sama.
		var existingUser model.User

		err = tx.
			Where("LOWER(email) = ?", ownerEmail).
			First(&existingUser).Error

		if err == nil {
			return errors.New("OWNER email is already registered")
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check OWNER email: %w", err)
		}

		passwordHash, err := password.Hash(ownerPassword)
		if err != nil {
			return fmt.Errorf("hash OWNER password: %w", err)
		}

		userID, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate OWNER UUID: %w", err)
		}

		owner := model.User{
			ID:           userID,
			FullName:     ownerFullName,
			Email:        ownerEmail,
			PasswordHash: passwordHash,
			Role:         model.RoleOwner,
		}

		if err := tx.Create(&owner).Error; err != nil {
			return fmt.Errorf("create OWNER: %w", err)
		}

		created = true
		return nil
	})

	if err != nil {
		return fmt.Errorf("seed OWNER failed: %w", err)
	}

	result := "already_exists"
	if created {
		result = "created"
	}
	logger.Info("owner seed completed", "operation", "seed_owner", "result", result)

	return nil
}
