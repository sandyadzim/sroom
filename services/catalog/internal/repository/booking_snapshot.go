package repository

import (
	"context"
	"errors"

	"catalog/internal/model"

	"gorm.io/gorm"
)

func (r *CarRepository) DemoHoldsEnabled() bool {
	return r.demoHolds
}

func (r *CarRepository) BookingSnapshot(ctx context.Context, id string) (model.Car, []string, error) {
	var car model.Car
	unitIDs := []string{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Select("id", "brand_id", "model", "trim", "daily_rate_idr", "status").
			Preload("Brand").Take(&car, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCarNotFound
		}
		if err != nil || car.Status != model.CarActive {
			return err
		}
		return tx.Model(&model.CarUnit{}).Where("car_id = ? AND status = ?", id, model.UnitActive).
			Order("id").Pluck("id", &unitIDs).Error
	}, &carReadOptions)
	return car, unitIDs, err
}
