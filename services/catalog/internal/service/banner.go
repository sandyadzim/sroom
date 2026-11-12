package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"
)

var ErrInvalidBanner = errors.New("check the banner content fields")

type BannerService struct {
	repo               *repository.BannerRepository
	images             ImageChecker
	publicImageBaseURL string
}

func NewBannerService(repo *repository.BannerRepository, images ImageChecker, publicImageBaseURL string) *BannerService {
	return &BannerService{repo: repo, images: images, publicImageBaseURL: strings.TrimRight(publicImageBaseURL, "/")}
}

func (s *BannerService) List(ctx context.Context) ([]dto.BannerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]dto.BannerResponse, 0, len(items))
	for _, item := range items {
		data = append(data, s.bannerResponse(item))
	}
	return data, nil
}

func (s *BannerService) Detail(ctx context.Context, position int) (dto.BannerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.FindByPosition(ctx, position)
	if err != nil {
		return dto.BannerResponse{}, err
	}
	return s.bannerResponse(item), nil
}

func (s *BannerService) Update(ctx context.Context, position int, req dto.UpdateBannerRequest) (dto.BannerResponse, error) {
	for _, field := range []*string{&req.CarModel, &req.CarSlug, &req.HeadlineLine1, &req.HeadlineLine2, &req.Description} {
		*field = strings.TrimSpace(*field)
		if strings.ContainsRune(*field, 0) {
			return dto.BannerResponse{}, ErrInvalidBanner
		}
	}
	if validate.Struct(req) != nil || !validCarSlug(req.CarSlug) {
		return dto.BannerResponse{}, ErrInvalidBanner
	}
	if err := checkImage(ctx, s.images, &req.ImageObjectKey); err != nil {
		return dto.BannerResponse{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Update(ctx, model.Banner{
		Position: position, BrandID: &req.BrandID, CarModel: &req.CarModel, CarSlug: &req.CarSlug,
		HeadlineLine1: &req.HeadlineLine1, HeadlineLine2: &req.HeadlineLine2,
		Description: &req.Description, ImageObjectKey: &req.ImageObjectKey,
	})
	if err != nil {
		return dto.BannerResponse{}, err
	}
	return s.bannerResponse(item), nil
}

func (s *BannerService) Delete(ctx context.Context, position int) (dto.BannerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	item, err := s.repo.Update(ctx, model.Banner{Position: position})
	if err != nil {
		return dto.BannerResponse{}, err
	}
	return s.bannerResponse(item), nil
}

func (s *BannerService) ListPublic(ctx context.Context) ([]dto.PublicBannerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	items, err := s.repo.ListFilled(ctx)
	if err != nil {
		return nil, err
	}
	data := make([]dto.PublicBannerResponse, 0, len(items))
	for _, item := range items {
		content := s.bannerResponse(item).Content
		data = append(data, dto.PublicBannerResponse{
			Slug: content.CarSlug, Make: content.BrandName, Model: content.CarModel,
			Image: content.ImageURL, Headline: [2]string{content.HeadlineLine1, content.HeadlineLine2},
			Description: content.Description,
		})
	}
	return data, nil
}

func (s *BannerService) bannerResponse(item model.Banner) dto.BannerResponse {
	result := dto.BannerResponse{Position: item.Position}
	if item.BrandID == nil {
		return result
	}
	result.Content = &dto.BannerContent{
		BrandID: *item.BrandID, BrandName: item.Brand.Name,
		CarModel: *item.CarModel, CarSlug: *item.CarSlug,
		HeadlineLine1: *item.HeadlineLine1, HeadlineLine2: *item.HeadlineLine2,
		Description: *item.Description, ImageObjectKey: *item.ImageObjectKey,
	}
	if s.publicImageBaseURL != "" {
		imageURL := s.publicImageBaseURL + "/" + *item.ImageObjectKey
		result.Content.ImageURL = &imageURL
	}
	return result
}

func validCarSlug(slug string) bool {
	for _, part := range strings.Split(slug, "-") {
		if part == "" || strings.Trim(part, "abcdefghijklmnopqrstuvwxyz0123456789") != "" {
			return false
		}
	}
	return true
}
