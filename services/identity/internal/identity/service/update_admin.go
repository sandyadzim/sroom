package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
)

type UpdateAdminRepository interface {
	UpdateAdmin(ctx context.Context, id uuid.UUID, changes map[string]any) (*model.User, error)
}

type UpdateAdminService struct {
	userRepo UpdateAdminRepository
}

func NewUpdateAdminService(userRepo UpdateAdminRepository) *UpdateAdminService {
	return &UpdateAdminService{userRepo: userRepo}
}

func (s *UpdateAdminService) Update(ctx context.Context, id uuid.UUID, req dto.UpdateAdminRequest) (*dto.AdminResponse, error) {
	if req.FullName != nil {
		name := strings.TrimSpace(*req.FullName)
		req.FullName = &name
	}
	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		req.Email = &email
	}
	if err := dto.ValidateUpdateAdmin(req); err != nil {
		return nil, err
	}
	changes := make(map[string]any)
	if req.FullName != nil {
		changes["full_name"] = *req.FullName
	}
	if req.Email != nil {
		changes["email"] = *req.Email
	}
	if req.Status != nil {
		changes["status"] = *req.Status
	}
	if req.Password != nil {
		hash, err := password.Hash(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("hash admin password: %w", err)
		}
		changes["password_hash"] = hash
	}
	user, err := s.userRepo.UpdateAdmin(ctx, id, changes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("update admin: %w", err)
	}
	result := adminResponse(*user)
	return &result, nil
}
