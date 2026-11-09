package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"identity/internal/identity/model"
	"identity/internal/identity/password"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load .env jika tersedia.
	// Environment variable yang sudah ada tidak ditimpa.
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

	if len(ownerFullName) > 100 {
		return errors.New("OWNER_FULL_NAME must be at most 100 characters")
	}

	if len(ownerEmail) > 255 {
		return errors.New("OWNER_EMAIL must be at most 255 characters")
	}

	if len(ownerPassword) < 8 {
		return errors.New("OWNER_PASSWORD must be at least 8 bytes")
	}

	if len(ownerPassword) > 72 {
		return errors.New("OWNER_PASSWORD must be at most 72 bytes")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		TranslateError: true,
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
		// Mencegah dua proses seeder berjalan bersamaan
		// dan sama-sama mencoba membuat OWNER.
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
			log.Printf(
				"OWNER already exists: %s. No changes made.",
				existingOwner.Email,
			)
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
			return fmt.Errorf(
				"email %q is already registered with role %s",
				ownerEmail,
				existingUser.Role,
			)
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

	if created {
		log.Printf("OWNER created successfully: %s", ownerEmail)
	}

	return nil
}
