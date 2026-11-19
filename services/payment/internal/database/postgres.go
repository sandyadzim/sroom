package database

import (
	"errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	// Readiness checks connectivity; a temporary outage must not stop HTTP startup.
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		return nil, errors.New("could not open PostgreSQL with the supplied DATABASE_URL")
	}
	return db, nil
}
