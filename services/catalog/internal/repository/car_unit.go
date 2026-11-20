package repository

import (
	"context"
	"errors"

	"catalog/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCarUnitNotFound  = errors.New("car unit not found")
	ErrCarUnitCodeTaken = errors.New("unit code already exists")
)

type CarUnitRepository struct {
	db *gorm.DB
}

func NewCarUnitRepository(db *gorm.DB) *CarUnitRepository {
	return &CarUnitRepository{db: db}
}

func (r *CarUnitRepository) List(ctx context.Context, carID, status string, offset, limit int) ([]model.CarUnit, int64, error) {
	var units []model.CarUnit
	var total int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var car model.Car
		err := tx.Select("id").First(&car, "id = ?", carID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCarNotFound
		}
		if err != nil {
			return err
		}

		query := tx.Model(&model.CarUnit{}).Where("car_id = ?", carID)
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if err := query.Count(&total).Error; err != nil {
			return err
		}
		return query.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&units).Error
	}, &carReadOptions)
	return units, total, err
}

func (r *CarUnitRepository) FindByID(ctx context.Context, id string) (model.CarUnit, error) {
	var unit model.CarUnit
	err := r.db.WithContext(ctx).First(&unit, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CarUnit{}, ErrCarUnitNotFound
	}
	return unit, err
}

func (r *CarUnitRepository) Create(ctx context.Context, unit model.CarUnit) (model.CarUnit, error) {
	err := r.db.WithContext(ctx).Create(&unit).Error
	return unit, carUnitWriteError(err)
}

func (r *CarUnitRepository) Update(ctx context.Context, id string, changes map[string]any) (model.CarUnit, error) {
	db := r.db.WithContext(ctx)
	var unit model.CarUnit
	result := db.Model(&unit).Where("id = ?", id).
		Clauses(clause.Returning{}).Select("internal_code", "status", "updated_at").Updates(changes)
	if result.Error != nil {
		return model.CarUnit{}, carUnitWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return model.CarUnit{}, ErrCarUnitNotFound
	}
	return unit, nil
}

func carUnitWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.ConstraintName {
		case "car_units_car_id_fkey":
			return ErrCarNotFound
		case "car_units_internal_code_unique_idx":
			return ErrCarUnitCodeTaken
		}
	}
	return err
}
