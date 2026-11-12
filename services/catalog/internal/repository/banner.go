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
	ErrBannerNotFound = errors.New("banner slot not found")
	ErrBannerBrand    = errors.New("brand not found")
)

type BannerRepository struct {
	db *gorm.DB
}

func NewBannerRepository(db *gorm.DB) *BannerRepository {
	return &BannerRepository{db: db}
}

func (r *BannerRepository) List(ctx context.Context) ([]model.Banner, error) {
	var banners []model.Banner
	err := r.db.WithContext(ctx).Preload("Brand").Order("position").Find(&banners).Error
	return banners, err
}

func (r *BannerRepository) FindByPosition(ctx context.Context, position int) (model.Banner, error) {
	var banner model.Banner
	err := r.db.WithContext(ctx).Preload("Brand").First(&banner, "position = ?", position).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Banner{}, ErrBannerNotFound
	}
	return banner, err
}

func (r *BannerRepository) ListFilled(ctx context.Context) ([]model.Banner, error) {
	var banners []model.Banner
	err := r.db.WithContext(ctx).Preload("Brand").Where("brand_id IS NOT NULL").
		Order("position").Find(&banners).Error
	return banners, err
}

func (r *BannerRepository) Update(ctx context.Context, banner model.Banner) (model.Banner, error) {
	var saved model.Banner
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&saved).Where("position = ?", banner.Position).
			Clauses(clause.Returning{}).
			Select("brand_id", "car_model", "car_slug", "headline_line_1", "headline_line_2", "description", "image_object_key").
			Updates(banner)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrBannerNotFound
		}
		if saved.BrandID != nil {
			var brand model.Brand
			if err := tx.First(&brand, "id = ?", *saved.BrandID).Error; err != nil {
				return err
			}
			saved.Brand = &brand
		}
		return nil
	})
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.ConstraintName == "banners_brand_id_fkey" {
		return model.Banner{}, ErrBannerBrand
	}
	return saved, err
}
