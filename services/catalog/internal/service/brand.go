package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var (
	ErrInvalidBrandName     = errors.New("name must contain 1 to 100 characters")
	ErrInvalidBrandStatus   = errors.New("status must be active or inactive")
	ErrEmptyUpdate          = errors.New("send at least one field to update")
	ErrInvalidImageFilename = errors.New("image_filename must be a file name of up to 255 characters and requires image_object_key")
)

type BrandService struct {
	repo               *repository.BrandRepository
	images             ImageChecker
	publicImageBaseURL string
}

func NewBrandService(repo *repository.BrandRepository, images ImageChecker, publicImageBaseURL string) *BrandService {
	return &BrandService{repo: repo, images: images, publicImageBaseURL: strings.TrimRight(publicImageBaseURL, "/")}
}

func (s *BrandService) List(ctx context.Context, search string, page, limit int) (*dto.BrandListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	offset := (page - 1) * limit
	items, total, err := s.repo.List(ctx, search, offset, limit)
	if err != nil {
		return nil, err
	}
	data := make([]dto.BrandResponse, 0, len(items))
	for _, item := range items {
		data = append(data, s.brandResponse(item))
	}
	return &dto.BrandListResponse{
		Data: data,
		Page: dto.BrandPage{Number: page, Limit: limit, Total: total},
	}, nil
}

func (s *BrandService) Create(ctx context.Context, req dto.CreateBrandRequest) (*dto.BrandResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := validateCreateBrand(req); err != nil {
		return nil, err
	}
	if err := validateBrandImageFilename(req.ImageObjectKey, req.ImageFilename); err != nil {
		return nil, err
	}
	if err := checkImage(ctx, s.images, req.ImageObjectKey); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Create(ctx, model.Brand{
		Name: req.Name, ImageObjectKey: req.ImageObjectKey, ImageFilename: req.ImageFilename,
	})
	if err != nil {
		return nil, err
	}
	result := s.brandResponse(item)
	return &result, nil
}

func (s *BrandService) Update(ctx context.Context, id string, changes model.BrandChanges) (*dto.BrandResponse, error) {
	if err := validateBrandImageFilename(changes.ImageObjectKey, changes.ImageFilename); err != nil {
		return nil, err
	}
	if changes.Name == nil && !changes.SetImage && changes.Status == nil {
		return nil, ErrEmptyUpdate
	}
	if changes.Name != nil {
		trimmedName := strings.TrimSpace(*changes.Name)
		if err := validateCreateBrand(dto.CreateBrandRequest{Name: trimmedName}); err != nil {
			return nil, err
		}
		changes.Name = &trimmedName
	}
	if changes.Status != nil && *changes.Status != model.BrandActive && *changes.Status != model.BrandInactive {
		return nil, ErrInvalidBrandStatus
	}
	if err := checkImage(ctx, s.images, changes.ImageObjectKey); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Update(ctx, id, changes)
	if err != nil {
		return nil, err
	}
	result := s.brandResponse(item)
	return &result, nil
}

func validateCreateBrand(req dto.CreateBrandRequest) error {
	if strings.ContainsRune(req.Name, 0) || validate.Struct(req) != nil {
		return ErrInvalidBrandName
	}
	return nil
}

func validateBrandImageFilename(key, filename *string) error {
	if filename == nil {
		return nil
	}
	if key == nil || strings.TrimSpace(*filename) == "" || utf8.RuneCountInString(*filename) > 255 ||
		strings.ContainsAny(*filename, "/\\") || strings.IndexFunc(*filename, unicode.IsControl) >= 0 {
		return ErrInvalidImageFilename
	}
	return nil
}

func (s *BrandService) brandResponse(item model.Brand) dto.BrandResponse {
	return dto.BrandResponse{
		ID:             item.ID,
		Name:           item.Name,
		ImageObjectKey: item.ImageObjectKey,
		ImageFilename:  item.ImageFilename,
		ImageURL:       brandImageURL(s.publicImageBaseURL, item.ImageObjectKey),
		Status:         item.Status,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func brandImageURL(baseURL string, objectKey *string) *string {
	if baseURL == "" || objectKey == nil {
		return nil
	}
	imageURL := baseURL + "/" + *objectKey
	return &imageURL
}
