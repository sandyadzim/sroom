package model

import "time"

const (
	CarActive   = "active"
	CarInactive = "inactive"
	CarCover    = "cover"
	CarGallery  = "gallery"
)

type Car struct {
	ID           string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BrandID      string `gorm:"type:uuid"`
	Brand        Brand
	Model        string
	Trim         string
	Slug         string
	Category     string
	BodyType     string
	Seats        int
	Transmission string
	FuelType     string
	Description  string
	DailyRateIDR int64 `gorm:"column:daily_rate_idr"`
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Images       []CarImage
}

type CarImage struct {
	ID        string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CarID     string `gorm:"type:uuid"`
	ObjectKey string
	Kind      string
	AltText   string
	Label     string
	Position  int
	CreatedAt time.Time
}

type CarFilter struct {
	Search       string
	BrandID      string
	Seats        int
	MinPrice     *int64
	MaxPrice     *int64
	Category     string
	Transmission string
	Status       string
	Sort         string
	Page         int
	Limit        int
}
