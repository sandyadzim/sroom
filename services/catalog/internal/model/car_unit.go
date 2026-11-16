package model

import "time"

const (
	UnitActive   = "active"
	UnitInactive = "inactive"
)

type CarUnit struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CarID        string `gorm:"type:uuid"`
	InternalCode string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
