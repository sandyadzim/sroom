package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
    UserID    uuid.UUID
    TokenHash string    `json:"-"`
    ExpiresAt time.Time
    CreatedAt time.Time
}
