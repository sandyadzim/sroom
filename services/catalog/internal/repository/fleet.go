package repository

import (
	"context"

	"catalog/internal/dto"

	"gorm.io/gorm"
)

func (r *CarRepository) FleetSummary(ctx context.Context) (dto.FleetSummary, error) {
	var result dto.FleetSummary
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(`SELECT count(*) AS total,
   count(*) FILTER (WHERE status = 'active') AS active,
   count(*) FILTER (WHERE status = 'inactive') AS inactive FROM cars`).Scan(&result.Cars).Error; err != nil {
			return err
		}
		return tx.Raw(`SELECT count(*) AS total,
   count(*) FILTER (WHERE status = 'active') AS active,
   count(*) FILTER (WHERE status = 'inactive') AS inactive FROM car_units`).Scan(&result.Units).Error
	}, &carReadOptions)
	return result, err
}
