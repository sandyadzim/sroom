package repository

import (
	"context"
	"errors"

	"catalog/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
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
	err := withBannerCar(r.db.WithContext(ctx)).Order("position").Find(&banners).Error
	return banners, err
}

func (r *BannerRepository) FindByPosition(ctx context.Context, position int) (model.Banner, error) {
	var banner model.Banner
	err := withBannerCar(r.db.WithContext(ctx)).First(&banner, "position = ?", position).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Banner{}, ErrBannerNotFound
	}
	return banner, err
}

func (r *BannerRepository) ListFilled(ctx context.Context) ([]model.Banner, error) {
	var banners []model.Banner
	err := withBannerCar(r.db.WithContext(ctx)).Where("brand_id IS NOT NULL").
		Order("position").Find(&banners).Error
	return banners, err
}

func withBannerCar(db *gorm.DB) *gorm.DB {
	return db.Preload("Brand").Preload("Car.Brand")
}

func (r *BannerRepository) Update(ctx context.Context, banner model.Banner) (model.Banner, error) {
	var saved model.Banner
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := setBannerCar(tx, &banner); err != nil {
			return err
		}
		result := tx.Model(&model.Banner{}).Where("position = ?", banner.Position).
			Select("car_id", "brand_id", "car_model", "car_slug", "headline_line_1", "headline_line_2", "description", "image_object_key").
			Updates(banner)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrBannerNotFound
		}
		return withBannerCar(tx).First(&saved, "position = ?", banner.Position).Error
	})
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.ConstraintName == "banners_brand_id_fkey" {
		return model.Banner{}, ErrBannerBrand
	}
	return saved, err
}

// Keep the old fields populated for existing clients, including after a rollback.
func setBannerCar(tx *gorm.DB, banner *model.Banner) error {
	var car model.Car
	if banner.CarID != nil {
		err := tx.First(&car, "id = ?", *banner.CarID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCarNotFound
		}
		if err != nil {
			return err
		}
		banner.BrandID, banner.CarModel, banner.CarSlug = &car.BrandID, &car.Model, &car.Slug
	} else if banner.BrandID != nil {
		// A save from the existing form can link a Car using its brand and slug.
		err := tx.Select("id").First(&car, "brand_id = ? AND slug = ?", *banner.BrandID, *banner.CarSlug).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		banner.CarID = &car.ID
	}
	return nil
}
