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

var ErrAdminNotFound = errors.New("admin not found")

type ReadAdminRepository interface {
	ListAdmins(ctx context.Context, limit, offset int, search, status string) ([]model.User, int64, error)
	FindAdminByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type ReadAdminService struct {
	userRepo ReadAdminRepository
}

func NewReadAdminService(userRepo ReadAdminRepository) *ReadAdminService {
	return &ReadAdminService{userRepo: userRepo}
}

func (s *ReadAdminService) List(ctx context.Context, page, limit int, search, status string) (*dto.AdminListResponse, error) {
	users, total, err := s.userRepo.ListAdmins(ctx, limit, (page-1)*limit, search, status)
	if err != nil {
		return nil, fmt.Errorf("list admins: %w", err)
	}

	data := make([]dto.AdminResponse, 0, len(users))
	for _, user := range users {
		data = append(data, adminResponse(user))
	}
	return &dto.AdminListResponse{
		Data: data,
		Page: dto.AccountPage{Number: page, Limit: limit, Total: total},
	}, nil
}

func (s *ReadAdminService) Detail(ctx context.Context, id uuid.UUID) (*dto.AdminResponse, error) {
	user, err := s.userRepo.FindAdminByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("find admin: %w", err)
	}
	result := adminResponse(*user)
	return &result, nil
}

func adminResponse(user model.User) dto.AdminResponse {
	return dto.AdminResponse{
		ID:        user.ID.String(),
		FullName:  user.FullName,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
