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

type CreateAdminService struct {
	userRepo UserRepository
}

func NewCreateAdminService(userRepo UserRepository) *CreateAdminService {
	return &CreateAdminService{userRepo: userRepo}
}

func (s *CreateAdminService) Create(ctx context.Context, req dto.CreateAdminRequest) (*dto.AdminResponse, error) {
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := dto.ValidateCreateAdmin(req); err != nil {
		return nil, err
	}
	_, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find admin email: %w", err)
	}
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash admin password: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate admin ID: %w", err)
	}
	status := model.StatusActive
	if req.Status != nil {
		status = *req.Status
	}
	user := &model.User{ID: id, FullName: req.FullName, Email: req.Email, PasswordHash: hash, Role: model.RoleAdmin, Status: status}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create admin: %w", err)
	}
	result := adminResponse(*user)
	return &result, nil
}
