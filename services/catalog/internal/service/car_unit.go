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
)

var (
	ErrInvalidCarUnitCode   = errors.New("internal_code must contain 1 to 100 characters without control characters")
	ErrInvalidCarUnitStatus = errors.New("status must be active or inactive")
)

type CarUnitService struct {
	repo *repository.CarUnitRepository
}

func NewCarUnitService(repo *repository.CarUnitRepository) *CarUnitService {
	return &CarUnitService{repo: repo}
}

func (s *CarUnitService) List(ctx context.Context, carID, status string, page, limit int) (dto.CarUnitListResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	units, total, err := s.repo.List(ctx, carID, status, (page-1)*limit, limit)
	if err != nil {
		return dto.CarUnitListResponse{}, err
	}
	data := make([]dto.CarUnitResponse, 0, len(units))
	for _, unit := range units {
		data = append(data, carUnitResponse(unit))
	}
	return dto.CarUnitListResponse{
		Data: data, Page: dto.CarPage{Number: page, Limit: limit, Total: total},
	}, nil
}

func (s *CarUnitService) Detail(ctx context.Context, id string) (dto.CarUnitResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	unit, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.CarUnitResponse{}, err
	}
	return carUnitResponse(unit), nil
}

func (s *CarUnitService) Create(ctx context.Context, carID string, req dto.CreateCarUnitRequest) (dto.CarUnitResponse, error) {
	code, err := cleanUnitCode(req.InternalCode)
	if err != nil {
		return dto.CarUnitResponse{}, err
	}
	status := model.UnitInactive
	if req.Status != nil {
		status = *req.Status
	}
	if !validUnitStatus(status) {
		return dto.CarUnitResponse{}, ErrInvalidCarUnitStatus
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	unit, err := s.repo.Create(ctx, model.CarUnit{CarID: carID, InternalCode: code, Status: status})
	if err != nil {
		return dto.CarUnitResponse{}, err
	}
	return carUnitResponse(unit), nil
}

func (s *CarUnitService) Update(ctx context.Context, id string, req dto.UpdateCarUnitRequest) (dto.CarUnitResponse, error) {
	changes := make(map[string]any)
	if req.InternalCode != nil {
		code, err := cleanUnitCode(*req.InternalCode)
		if err != nil {
			return dto.CarUnitResponse{}, err
		}
		changes["internal_code"] = code
	}
	if req.Status != nil {
		if !validUnitStatus(*req.Status) {
			return dto.CarUnitResponse{}, ErrInvalidCarUnitStatus
		}
		changes["status"] = *req.Status
	}
	if len(changes) == 0 {
		return dto.CarUnitResponse{}, ErrEmptyUpdate
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	unit, err := s.repo.Update(ctx, id, changes)
	if err != nil {
		return dto.CarUnitResponse{}, err
	}
	return carUnitResponse(unit), nil
}

func cleanUnitCode(value string) (string, error) {
	code := strings.TrimSpace(value)
	if code == "" || utf8.RuneCountInString(code) > 100 || strings.IndexFunc(code, unicode.IsControl) >= 0 {
		return "", ErrInvalidCarUnitCode
	}
	return code, nil
}

func validUnitStatus(status string) bool {
	return status == model.UnitActive || status == model.UnitInactive
}

func carUnitResponse(unit model.CarUnit) dto.CarUnitResponse {
	return dto.CarUnitResponse{
		ID: unit.ID, CarID: unit.CarID, InternalCode: unit.InternalCode, Status: unit.Status,
		CreatedAt: unit.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: unit.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
