package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
    ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
    FullName     string
    Email        string
    PasswordHash string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
