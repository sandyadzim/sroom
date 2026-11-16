package model

type FeaturedCar struct {
	Position int    `gorm:"primaryKey;autoIncrement:false"`
	CarID    string `gorm:"type:uuid"`
}

type FeaturedCarSelection struct {
	Car
	Position int
}
