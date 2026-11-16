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

const maxFeaturedCars = 3

var ErrInvalidFeaturedCars = errors.New("car_ids must contain up to three slots with distinct car IDs or null")

type FeaturedCarService struct {
	repo               *repository.FeaturedCarRepository
	publicImageBaseURL string
}

func NewFeaturedCarService(repo *repository.FeaturedCarRepository, publicImageBaseURL string) *FeaturedCarService {
	return &FeaturedCarService{repo: repo, publicImageBaseURL: strings.TrimRight(publicImageBaseURL, "/")}
}

func (s *FeaturedCarService) List(ctx context.Context) ([]dto.AdminFeaturedCar, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cars, err := s.repo.List(ctx, false)
	if err != nil {
		return nil, err
	}
	return s.adminCards(cars), nil
}

func (s *FeaturedCarService) Replace(ctx context.Context, req dto.UpdateFeaturedCarsRequest) ([]dto.AdminFeaturedCar, error) {
	if req.CarIDs == nil || len(req.CarIDs) > maxFeaturedCars {
		return nil, ErrInvalidFeaturedCars
	}
	selections := make([]model.FeaturedCar, 0, len(req.CarIDs))
	seen := make(map[uuid.UUID]bool, len(req.CarIDs))
	for index, rawID := range req.CarIDs {
		if rawID == nil {
			continue
		}
		id, err := uuid.Parse(*rawID)
		if len(*rawID) != 36 || err != nil || seen[id] {
			return nil, ErrInvalidFeaturedCars
		}
		seen[id] = true
		selections = append(selections, model.FeaturedCar{Position: index + 1, CarID: id.String()})
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cars, err := s.repo.Replace(ctx, selections)
	if err != nil {
		return nil, err
	}
	return s.adminCards(cars), nil
}

func (s *FeaturedCarService) ListPublic(ctx context.Context) ([]dto.CarCard, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cars, err := s.repo.List(ctx, true)
	if err != nil {
		return nil, err
	}
	data := make([]dto.CarCard, 0, len(cars))
	for _, car := range cars {
		data = append(data, carCard(car.Car, s.publicImageBaseURL))
	}
	return data, nil
}

func (s *FeaturedCarService) adminCards(cars []model.FeaturedCarSelection) []dto.AdminFeaturedCar {
	data := make([]dto.AdminFeaturedCar, 0, len(cars))
	for _, car := range cars {
		data = append(data, dto.AdminFeaturedCar{
			CarCard: carCard(car.Car, s.publicImageBaseURL),
			Position: car.Position,
			Status:  car.Status,
		})
	}
	return data
}
