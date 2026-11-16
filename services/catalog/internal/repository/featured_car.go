package repository

import (
	"context"

	"catalog/internal/model"

	"gorm.io/gorm"
)

type FeaturedCarRepository struct {
	db *gorm.DB
}

func NewFeaturedCarRepository(db *gorm.DB) *FeaturedCarRepository {
	return &FeaturedCarRepository{db: db}
}

func (r *FeaturedCarRepository) List(ctx context.Context, activeOnly bool) ([]model.FeaturedCarSelection, error) {
	var cars []model.FeaturedCarSelection
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		cars, err = listFeaturedCars(tx, activeOnly)
		return err
	}, &carReadOptions)
	return cars, err
}

func (r *FeaturedCarRepository) Replace(ctx context.Context, selections []model.FeaturedCar) ([]model.FeaturedCarSelection, error) {
	var cars []model.FeaturedCarSelection
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize replacements, including when the table is empty.
		if err := tx.Exec("LOCK TABLE featured_cars IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}

		if len(selections) > 0 {
			carIDs := make([]string, 0, len(selections))
			for _, selection := range selections {
				carIDs = append(carIDs, selection.CarID)
			}
			var count int64
			if err := tx.Model(&model.Car{}).Where("id IN ?", carIDs).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(carIDs)) {
				return ErrCarNotFound
			}
		}

		if err := tx.Exec("DELETE FROM featured_cars").Error; err != nil {
			return err
		}
		if len(selections) > 0 {
			if err := tx.Create(&selections).Error; err != nil {
				return err
			}
		}

		var err error
		cars, err = listFeaturedCars(tx, false)
		return err
	})
	return cars, err
}

func listFeaturedCars(db *gorm.DB, activeOnly bool) ([]model.FeaturedCarSelection, error) {
	query := withCarCover(db.Model(&model.Car{})).
		Select("cars.*, featured_cars.position").
		Joins("JOIN featured_cars ON featured_cars.car_id = cars.id").
		Order("featured_cars.position")
	if activeOnly {
		query = query.Where("cars.status = ?", model.CarActive)
	}
	var cars []model.FeaturedCarSelection
	err := query.Find(&cars).Error
	return cars, err
}
