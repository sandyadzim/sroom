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
	ErrBrandNotFound  = errors.New("brand not found")
	ErrBrandNameTaken = errors.New("brand name already exists")
)

type BrandRepository struct {
	db *gorm.DB
}

func NewBrandRepository(db *gorm.DB) *BrandRepository {
	return &BrandRepository{db: db}
}

func (r *BrandRepository) List(ctx context.Context, search string, offset, limit int) ([]model.Brand, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Brand{})
	if search != "" {
		query = query.Where("strpos(lower(name), lower(?)) > 0", search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var brands []model.Brand
	err := query.Order("LOWER(name), id").Offset(offset).Limit(limit).Find(&brands).Error
	return brands, total, err
}

func (r *BrandRepository) Create(ctx context.Context, item model.Brand) (model.Brand, error) {
	err := r.db.WithContext(ctx).Create(&item).Error
	return item, brandWriteError(err)
}

func (r *BrandRepository) Update(ctx context.Context, id string, input model.BrandChanges) (model.Brand, error) {
	changes := map[string]any{}
	if input.Name != nil {
		changes["name"] = *input.Name
	}
	if input.SetImage {
		changes["image_object_key"] = input.ImageObjectKey
	}
	if input.Status != nil {
		changes["status"] = *input.Status
	}

	var updated model.Brand
	// RETURNING reads the row saved by this update in the same query.
	result := r.db.WithContext(ctx).Model(&updated).Where("id = ?", id).
		Clauses(clause.Returning{}).Updates(changes)
	if result.Error != nil {
		return model.Brand{}, brandWriteError(result.Error)
	}
	if result.RowsAffected == 0 {
		return model.Brand{}, ErrBrandNotFound
	}
	return updated, nil
}

func brandWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" &&
		postgresError.ConstraintName == "brands_normalized_name_unique_idx" {
		return ErrBrandNameTaken
	}
	return err
}
