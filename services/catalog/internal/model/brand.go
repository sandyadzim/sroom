package model

import "time"

const (
	BrandActive   = "active"
	BrandInactive = "inactive"
)

type Brand struct {
	ID             string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string
	ImageObjectKey *string
	Status         string `gorm:"default:active"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type BrandChanges struct {
	Name           *string
	ImageObjectKey *string
	SetImage       bool
	Status         *string
}
