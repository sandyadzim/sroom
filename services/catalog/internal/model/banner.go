package model

type Banner struct {
	Position       int     `gorm:"primaryKey;autoIncrement:false"`
	BrandID        *string `gorm:"type:uuid"`
	Brand          *Brand
	CarModel       *string
	CarSlug        *string
	HeadlineLine1  *string `gorm:"column:headline_line_1"`
	HeadlineLine2  *string `gorm:"column:headline_line_2"`
	Description    *string
	ImageObjectKey *string
}
