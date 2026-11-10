package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName     string
	Email        string
	PasswordHash string `json:"-"`
	Role         Role
	Status       string `gorm:"default:active"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
