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

func (r *FeaturedCarRepository) List(ctx context.Context, activeOnly bool) ([]model.Car, error) {
	var cars []model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		cars, err = listFeaturedCars(tx, activeOnly)
		return err
	}, &carReadOptions)
	return cars, err
}

func (r *FeaturedCarRepository) Replace(ctx context.Context, carIDs []string) ([]model.Car, error) {
	var cars []model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize replacements, including when the table is empty.
		if err := tx.Exec("LOCK TABLE featured_cars IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}

		if len(carIDs) > 0 {
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
		if len(carIDs) > 0 {
			selections := make([]model.FeaturedCar, 0, len(carIDs))
			for index, id := range carIDs {
				selections = append(selections, model.FeaturedCar{Position: index + 1, CarID: id})
			}
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

func listFeaturedCars(db *gorm.DB, activeOnly bool) ([]model.Car, error) {
	query := withCarCover(db.Model(&model.Car{})).
		Select("cars.*").
		Joins("JOIN featured_cars ON featured_cars.car_id = cars.id").
		Order("featured_cars.position")
	if activeOnly {
		query = query.Where("cars.status = ?", model.CarActive)
	}
	var cars []model.Car
	err := query.Find(&cars).Error
	return cars, err
}
