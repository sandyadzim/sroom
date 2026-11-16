package repository

import (
	"context"
	"database/sql"
	"errors"

	"catalog/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var carReadOptions = sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}

var (
	ErrCarNotFound      = errors.New("car not found")
	ErrCarBrandInactive = errors.New("choose an active brand")
	ErrCarAlreadyExists = errors.New("car variant or slug already exists")
	ErrCarCoverRequired = errors.New("an active car requires a cover image")
)

type CarRepository struct {
	db *gorm.DB
}

func NewCarRepository(db *gorm.DB) *CarRepository {
	return &CarRepository{db: db}
}

func (r *CarRepository) List(ctx context.Context, filters model.CarFilter) ([]model.Car, int64, error) {
	var cars []model.Car
	var total int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		cars, total, err = listCars(tx, filters)
		return err
	}, &carReadOptions)
	return cars, total, err
}

func listCars(db *gorm.DB, filters model.CarFilter) ([]model.Car, int64, error) {
	query := db.Model(&model.Car{}).
		Joins("JOIN brands ON brands.id = cars.brand_id")
	if filters.Search != "" {
		query = query.Where(
			"(strpos(lower(brands.name), lower(?)) > 0 OR strpos(lower(cars.model), lower(?)) > 0 OR strpos(lower(cars.trim), lower(?)) > 0)",
			filters.Search, filters.Search, filters.Search,
		)
	}
	if filters.BrandID != "" {
		query = query.Where("cars.brand_id = ?", filters.BrandID)
	}
	if filters.Seats != 0 {
		query = query.Where("cars.seats = ?", filters.Seats)
	}
	if filters.MinPrice != nil {
		query = query.Where("cars.daily_rate_idr >= ?", *filters.MinPrice)
	}
	if filters.MaxPrice != nil {
		query = query.Where("cars.daily_rate_idr <= ?", *filters.MaxPrice)
	}
	if filters.Category != "" {
		query = query.Where("cars.category = ?", filters.Category)
	}
	if filters.Transmission != "" {
		query = query.Where("cars.transmission = ?", filters.Transmission)
	}
	if filters.Status != "" {
		query = query.Where("cars.status = ?", filters.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	switch filters.Sort {
	case "price-asc":
		query = query.Order("cars.daily_rate_idr ASC, cars.id ASC")
	case "price-desc":
		query = query.Order("cars.daily_rate_idr DESC, cars.id ASC")
	case "name-asc":
		query = query.Order("LOWER(brands.name), LOWER(cars.model), LOWER(cars.trim), cars.id")
	default:
		query = query.Order("cars.created_at DESC, cars.id DESC")
	}

	var cars []model.Car
	err := withCarCover(query).Offset((filters.Page - 1) * filters.Limit).Limit(filters.Limit).Find(&cars).Error
	return cars, total, err
}

func (r *CarRepository) FindByID(ctx context.Context, id string) (model.Car, error) {
	return findCarDetail(r.db.WithContext(ctx).Where("id = ?", id))
}

func publicCarBySlug(db *gorm.DB, slug string) *gorm.DB {
	return db.Where("cars.slug = ?", slug).
		Where("(cars.status = ? OR EXISTS (SELECT 1 FROM banners WHERE banners.car_id = cars.id))", model.CarActive)
}

func (r *CarRepository) FindPublic(ctx context.Context, slug string) (model.Car, error) {
	var car model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		car, err = findCarDetail(publicCarBySlug(tx, slug))
		return err
	}, &carReadOptions)
	return car, err
}

func (r *CarRepository) ListRelated(ctx context.Context, slug string, limit int) ([]model.Car, error) {
	var cars []model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source model.Car
		err := publicCarBySlug(tx, slug).Select("id").First(&source).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCarNotFound
		}
		if err != nil {
			return err
		}
		return withCarCover(tx).Where("status = ? AND id <> ?", model.CarActive, source.ID).
			Order("created_at DESC, id DESC").Limit(limit).Find(&cars).Error
	}, &carReadOptions)
	return cars, err
}

func (r *CarRepository) ListPublicBrands(ctx context.Context) ([]model.Brand, error) {
	var brands []model.Brand
	err := r.db.WithContext(ctx).Select("id", "name").
		Where("EXISTS (SELECT 1 FROM cars WHERE cars.brand_id = brands.id AND cars.status = ?)", model.CarActive).
		Order("LOWER(name), id").Find(&brands).Error
	return brands, err
}

func (r *CarRepository) ActiveBrand(ctx context.Context, id string) (model.Brand, error) {
	return findActiveCarBrand(r.db.WithContext(ctx), id)
}

func (r *CarRepository) Create(ctx context.Context, car model.Car, images []model.CarImage) (model.Car, error) {
	var saved model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockActiveCarBrand(tx, car.BrandID); err != nil {
			return err
		}
		if err := tx.Omit("Brand", "Images").Create(&car).Error; err != nil {
			return err
		}
		if err := insertCarImages(tx, car.ID, images); err != nil {
			return err
		}
		var err error
		saved, err = findCarDetail(tx.Where("id = ?", car.ID))
		return err
	})
	return saved, carWriteError(err)
}

func (r *CarRepository) Update(ctx context.Context, id string, changes map[string]any, cover *model.CarImage, gallery []model.CarImage) (model.Car, error) {
	var saved model.Car
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		car, err := lockCar(tx, id)
		if err != nil {
			return err
		}
		if brandID, ok := changes["brand_id"].(string); ok && brandID != car.BrandID {
			if _, err := lockActiveCarBrand(tx, brandID); err != nil {
				return err
			}
		}
		if changes["status"] == model.CarActive && cover == nil {
			var count int64
			err := tx.Model(&model.CarImage{}).Where("car_id = ? AND kind = ?", id, model.CarCover).Count(&count).Error
			if err != nil {
				return err
			}
			if count == 0 {
				return ErrCarCoverRequired
			}
		}
		if cover != nil {
			if err := replaceCarImages(tx, id, model.CarCover, []model.CarImage{*cover}); err != nil {
				return err
			}
		}
		if gallery != nil {
			if err := replaceCarImages(tx, id, model.CarGallery, gallery); err != nil {
				return err
			}
		}
		if len(changes) == 0 {
			changes = map[string]any{"updated_at": tx.NowFunc()}
		}
		if err := tx.Model(&car).Updates(changes).Error; err != nil {
			return err
		}
		saved, err = findCarDetail(tx.Where("id = ?", id))
		return err
	})
	return saved, carWriteError(err)
}

func replaceCarImages(tx *gorm.DB, carID, kind string, images []model.CarImage) error {
	if err := tx.Where("car_id = ? AND kind = ?", carID, kind).Delete(&model.CarImage{}).Error; err != nil {
		return err
	}
	return insertCarImages(tx, carID, images)
}

func insertCarImages(tx *gorm.DB, carID string, images []model.CarImage) error {
	for i := range images {
		images[i].CarID = carID
	}
	return tx.Create(&images).Error
}

func withCarCover(db *gorm.DB) *gorm.DB {
	return db.Preload("Brand").Preload("Images", "kind = ?", model.CarCover)
}

func findCarDetail(db *gorm.DB) (model.Car, error) {
	var car model.Car
	err := db.Preload("Brand").Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("kind, position")
	}).First(&car).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Car{}, ErrCarNotFound
	}
	return car, err
}

func lockCar(tx *gorm.DB, id string) (model.Car, error) {
	var car model.Car
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&car, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Car{}, ErrCarNotFound
	}
	return car, err
}

func lockActiveCarBrand(tx *gorm.DB, id string) (model.Brand, error) {
	return findActiveCarBrand(tx.Clauses(clause.Locking{Strength: "SHARE"}), id)
}

func findActiveCarBrand(db *gorm.DB, id string) (model.Brand, error) {
	var brand model.Brand
	err := db.First(&brand, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Brand{}, ErrBrandNotFound
	}
	if err == nil && brand.Status != model.BrandActive {
		return model.Brand{}, ErrCarBrandInactive
	}
	return brand, err
}

func carWriteError(err error) error {
	if postgresError, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch postgresError.ConstraintName {
		case "cars_slug_key", "cars_variant_unique_idx":
			return ErrCarAlreadyExists
		case "cars_brand_id_fkey":
			return ErrBrandNotFound
		}
	}
	return err
}
