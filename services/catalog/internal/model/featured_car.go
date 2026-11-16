package model

type FeaturedCar struct {
	Position int    `gorm:"primaryKey;autoIncrement:false"`
	CarID    string `gorm:"type:uuid"`
}
