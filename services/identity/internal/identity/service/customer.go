package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
)

var ErrCustomerNotFound = errors.New("customer not found")
var ErrInvalidCustomerStatus = errors.New("status must be active or inactive")

type CustomerRepository interface {
	ListCustomers(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error)
	FindCustomerByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateCustomerStatus(ctx context.Context, id uuid.UUID, status string) (*model.User, error)
}

type CustomerService struct {
	userRepo CustomerRepository
}

func NewCustomerService(userRepo CustomerRepository) *CustomerService {
	return &CustomerService{userRepo: userRepo}
}

func (s *CustomerService) List(ctx context.Context, page, limit int, search, status string) (*dto.CustomerListResponse, error) {
	users, total, err := s.userRepo.ListCustomers(ctx, limit, (page-1)*limit, search, status)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	customers := make([]dto.CustomerResponse, 0, len(users))
	for _, user := range users {
		customers = append(customers, customerResponse(user))
	}
	return &dto.CustomerListResponse{
		Data: customers,
		Page: dto.AccountPage{Number: page, Limit: limit, Total: total},
	}, nil
}

func (s *CustomerService) Detail(ctx context.Context, id uuid.UUID) (*dto.CustomerResponse, error) {
	user, err := s.userRepo.FindCustomerByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find customer: %w", err)
	}

	result := customerResponse(*user)
	return &result, nil
}

func (s *CustomerService) UpdateStatus(ctx context.Context, id uuid.UUID, req dto.UpdateCustomerStatusRequest) (*dto.CustomerResponse, error) {
	if req.Status != model.StatusActive && req.Status != model.StatusInactive {
		return nil, ErrInvalidCustomerStatus
	}

	user, err := s.userRepo.UpdateCustomerStatus(ctx, id, req.Status)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrCustomerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update customer status: %w", err)
	}

	result := customerResponse(*user)
	return &result, nil
}

func customerResponse(user model.User) dto.CustomerResponse {
	return dto.CustomerResponse{
		ID:        user.ID.String(),
		FullName:  user.FullName,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
