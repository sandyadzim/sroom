package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrInvalidCar      = errors.New("check the car fields")
	ErrInvalidCarImage = errors.New("check the car image fields")
)

type CarService struct {
	repo               *repository.CarRepository
	images             ImageChecker
	publicImageBaseURL string
}

func NewCarService(repo *repository.CarRepository, images ImageChecker, publicImageBaseURL string) *CarService {
	return &CarService{repo: repo, images: images, publicImageBaseURL: strings.TrimRight(publicImageBaseURL, "/")}
}

func (s *CarService) List(ctx context.Context, filters model.CarFilter) (dto.AdminCarList, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return dto.AdminCarList{}, err
	}
	data := make([]dto.AdminCarCard, 0, len(items))
	for _, item := range items {
		data = append(data, s.adminCarCard(item))
	}
	return dto.AdminCarList{Data: data, Page: carPage(filters, total)}, nil
}

func (s *CarService) Detail(ctx context.Context, id string) (dto.AdminCarDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	return s.adminCarDetail(item), nil
}

func (s *CarService) Create(ctx context.Context, req dto.CreateCarRequest) (dto.AdminCarDetail, error) {
	if !cleanCarText(&req.Model, &req.Trim, &req.Description) || validate.Struct(req) != nil {
		return dto.AdminCarDetail{}, ErrInvalidCar
	}
	cover, gallery, err := s.prepareCarPhotos(ctx, req.Cover, req.Gallery)
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	status := model.CarInactive
	if req.Status != nil {
		status = *req.Status
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	brand, err := s.repo.ActiveBrand(ctx, req.BrandID)
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	slug := makeCarSlug(brand.Name, req.Model, req.Trim)
	if slug == "" {
		return dto.AdminCarDetail{}, ErrInvalidCar
	}

	item, err := s.repo.Create(ctx, model.Car{
		BrandID: brand.ID, Model: req.Model, Trim: req.Trim, Slug: slug,
		Category: req.Category, BodyType: req.BodyType, Seats: req.Seats,
		Transmission: req.Transmission, FuelType: req.FuelType, Description: req.Description,
		DailyRateIDR: req.DailyRateIDR, Status: status,
	}, append([]model.CarImage{*cover}, gallery...))
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	return s.adminCarDetail(item), nil
}

func (s *CarService) Update(ctx context.Context, id string, req dto.UpdateCarRequest) (dto.AdminCarDetail, error) {
	if !cleanCarText(req.Model, req.Trim, req.Description) || validate.Struct(req) != nil {
		return dto.AdminCarDetail{}, ErrInvalidCar
	}
	changes := map[string]any{}
	if req.BrandID != nil {
		brandID, err := uuid.Parse(*req.BrandID)
		if err != nil {
			return dto.AdminCarDetail{}, ErrInvalidCar
		}
		changes["brand_id"] = brandID.String()
	}
	if req.Model != nil {
		changes["model"] = *req.Model
	}
	if req.Trim != nil {
		changes["trim"] = *req.Trim
	}
	if req.Category != nil {
		changes["category"] = *req.Category
	}
	if req.BodyType != nil {
		changes["body_type"] = *req.BodyType
	}
	if req.Seats != nil {
		changes["seats"] = *req.Seats
	}
	if req.Transmission != nil {
		changes["transmission"] = *req.Transmission
	}
	if req.FuelType != nil {
		changes["fuel_type"] = *req.FuelType
	}
	if req.Description != nil {
		changes["description"] = *req.Description
	}
	if req.DailyRateIDR != nil {
		changes["daily_rate_idr"] = *req.DailyRateIDR
	}
	if req.Status != nil {
		changes["status"] = *req.Status
	}
	if len(changes) == 0 && req.Cover == nil && req.Gallery == nil {
		return dto.AdminCarDetail{}, ErrEmptyUpdate
	}
	cover, gallery, err := s.prepareCarPhotos(ctx, req.Cover, req.Gallery)
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Update(ctx, id, changes, cover, gallery)
	if err != nil {
		return dto.AdminCarDetail{}, err
	}
	return s.adminCarDetail(item), nil
}

func (s *CarService) prepareCarPhotos(ctx context.Context, cover *dto.CarPhotoInput, gallery []dto.CarPhotoInput) (*model.CarImage, []model.CarImage, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var coverImage *model.CarImage
	if cover != nil {
		image, err := s.prepareCarImage(ctx, model.CarCover, *cover)
		if err != nil {
			return nil, nil, err
		}
		coverImage = &image
	}

	var galleryImages []model.CarImage
	for position, photo := range gallery {
		image, err := s.prepareCarImage(ctx, model.CarGallery, photo)
		if err != nil {
			return nil, nil, err
		}
		image.Position = position
		galleryImages = append(galleryImages, image)
	}
	return coverImage, galleryImages, nil
}

func (s *CarService) prepareCarImage(ctx context.Context, kind string, photo dto.CarPhotoInput) (model.CarImage, error) {
	if !cleanCarText(&photo.AltText, &photo.Label) || validate.Struct(photo) != nil {
		return model.CarImage{}, ErrInvalidCarImage
	}
	if err := checkImage(ctx, s.images, &photo.ObjectKey); err != nil {
		return model.CarImage{}, err
	}
	return model.CarImage{ObjectKey: photo.ObjectKey, Kind: kind, AltText: photo.AltText, Label: photo.Label}, nil
}

func (s *CarService) ListPublic(ctx context.Context, filters model.CarFilter) (dto.CarList, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	filters.Status = model.CarActive
	items, total, err := s.repo.List(ctx, filters)
	if err != nil {
		return dto.CarList{}, err
	}
	return dto.CarList{Data: s.carCards(items), Page: carPage(filters, total)}, nil
}

func (s *CarService) DetailPublic(ctx context.Context, slug string) (dto.CarDetail, error) {
	if !validCarSlug(slug) {
		return dto.CarDetail{}, repository.ErrCarNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.FindPublic(ctx, slug)
	if err != nil {
		return dto.CarDetail{}, err
	}
	gallery := make([]dto.CarImageResponse, 0, len(item.Images))
	for _, image := range item.Images {
		gallery = append(gallery, carImage(image, s.publicImageBaseURL))
	}
	return dto.CarDetail{
		CarCard: carCard(item, s.publicImageBaseURL), Status: item.Status, BodyType: item.BodyType, FuelType: item.FuelType,
		Description: item.Description, Gallery: gallery,
	}, nil
}

func (s *CarService) ListRelated(ctx context.Context, slug string, limit int) ([]dto.CarCard, error) {
	if !validCarSlug(slug) {
		return nil, repository.ErrCarNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, err := s.repo.ListRelated(ctx, slug, limit)
	if err != nil {
		return nil, err
	}
	return s.carCards(items), nil
}

func (s *CarService) ListPublicBrands(ctx context.Context) ([]dto.CarBrandOption, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	brands, err := s.repo.ListPublicBrands(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]dto.CarBrandOption, 0, len(brands))
	for _, brand := range brands {
		data = append(data, dto.CarBrandOption{ID: brand.ID, Name: brand.Name})
	}
	return data, nil
}

func (s *CarService) carCards(items []model.Car) []dto.CarCard {
	data := make([]dto.CarCard, 0, len(items))
	for _, item := range items {
		data = append(data, carCard(item, s.publicImageBaseURL))
	}
	return data
}

func carCard(item model.Car, publicImageBaseURL string) dto.CarCard {
	result := dto.CarCard{
		ID: item.ID, BrandID: item.BrandID, BrandName: item.Brand.Name,
		Model: item.Model, Trim: item.Trim, Slug: item.Slug, Category: item.Category,
		Seats: item.Seats, Transmission: item.Transmission, DailyRateIDR: item.DailyRateIDR,
	}
	for _, image := range item.Images {
		if image.Kind == model.CarCover {
			cover := carImage(image, publicImageBaseURL)
			result.Cover = &cover
			break
		}
	}
	return result
}

func (s *CarService) adminCarCard(item model.Car) dto.AdminCarCard {
	return dto.AdminCarCard{
		CarCard: carCard(item, s.publicImageBaseURL), Status: item.Status,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *CarService) adminCarDetail(item model.Car) dto.AdminCarDetail {
	images := make([]dto.AdminCarImageResponse, 0, len(item.Images))
	for _, image := range item.Images {
		images = append(images, s.adminCarImage(image))
	}
	return dto.AdminCarDetail{
		AdminCarCard: s.adminCarCard(item), BodyType: item.BodyType, FuelType: item.FuelType,
		Description: item.Description, Images: images,
	}
}

func (s *CarService) adminCarImage(image model.CarImage) dto.AdminCarImageResponse {
	return dto.AdminCarImageResponse{CarImageResponse: carImage(image, s.publicImageBaseURL), ID: image.ID, ObjectKey: image.ObjectKey, Kind: image.Kind}
}

func carImage(image model.CarImage, publicImageBaseURL string) dto.CarImageResponse {
	result := dto.CarImageResponse{AltText: image.AltText, Label: image.Label}
	if publicImageBaseURL != "" {
		imageURL := publicImageBaseURL + "/" + image.ObjectKey
		result.URL = &imageURL
	}
	return result
}

func carPage(filters model.CarFilter, total int64) dto.CarPage {
	return dto.CarPage{Number: filters.Page, Limit: filters.Limit, Total: total}
}

func cleanCarText(fields ...*string) bool {
	for _, field := range fields {
		if field == nil {
			continue
		}
		*field = strings.TrimSpace(*field)
		if strings.ContainsRune(*field, 0) {
			return false
		}
	}
	return true
}

func makeCarSlug(brand, name, trim string) string {
	parts := strings.FieldsFunc(strings.ToLower(brand+" "+name+" "+trim), func(char rune) bool {
		return (char < 'a' || char > 'z') && (char < '0' || char > '9')
	})
	return strings.Join(parts, "-")
}
