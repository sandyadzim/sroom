package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var (
	ErrInvalidBrandName   = errors.New("name must contain 1 to 100 characters")
	ErrInvalidBrandStatus = errors.New("status must be active or inactive")
	ErrEmptyUpdate        = errors.New("send at least one field to update")
	ErrInvalidImage       = errors.New("image_object_key must refer to an uploaded image")
	ErrImageUnavailable   = errors.New("image storage unavailable")
)

type ImageChecker interface {
	ImageExists(ctx context.Context, key string) (bool, error)
}

type BrandService struct {
	repo   *repository.BrandRepository
	images ImageChecker
}

func NewBrandService(repo *repository.BrandRepository, images ImageChecker) *BrandService {
	return &BrandService{repo: repo, images: images}
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
		data = append(data, brandResponse(item))
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
	if err := s.checkImage(ctx, req.ImageObjectKey); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Create(ctx, model.Brand{Name: req.Name, ImageObjectKey: req.ImageObjectKey})
	if err != nil {
		return nil, err
	}
	result := brandResponse(item)
	return &result, nil
}

func (s *BrandService) Update(ctx context.Context, id string, changes model.BrandChanges) (*dto.BrandResponse, error) {
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
	if err := s.checkImage(ctx, changes.ImageObjectKey); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Update(ctx, id, changes)
	if err != nil {
		return nil, err
	}
	result := brandResponse(item)
	return &result, nil
}

func validateCreateBrand(req dto.CreateBrandRequest) error {
	if strings.ContainsRune(req.Name, 0) || validate.Struct(req) != nil {
		return ErrInvalidBrandName
	}
	return nil
}

func brandResponse(item model.Brand) dto.BrandResponse {
	return dto.BrandResponse{
		ID:             item.ID,
		Name:           item.Name,
		ImageObjectKey: item.ImageObjectKey,
		Status:         item.Status,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *BrandService) checkImage(ctx context.Context, key *string) error {
	if key == nil {
		return nil
	}
	filename, validPrefix := strings.CutPrefix(*key, "catalog/images/")
	id, extension, hasExtension := strings.Cut(filename, ".")
	if !validPrefix || !hasExtension || len(id) != 26 ||
		strings.Trim(id, "abcdefghijklmnopqrstuvwxyz234567") != "" {
		return ErrInvalidImage
	}
	if extension != "jpeg" && extension != "png" && extension != "webp" {
		return ErrInvalidImage
	}
	if s.images == nil {
		return ErrImageUnavailable
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	exists, err := s.images.ImageExists(ctx, *key)
	if err != nil {
		return ErrImageUnavailable
	}
	if !exists {
		return ErrInvalidImage
	}
	return nil
}
