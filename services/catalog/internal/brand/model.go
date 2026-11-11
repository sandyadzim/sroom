package brand

import "time"

type Brand struct {
	ID             string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string
	ImageObjectKey *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
