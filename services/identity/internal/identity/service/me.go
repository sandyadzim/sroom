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

var ErrUserNotFound = errors.New("user not found")

type MeUserRepository interface {
	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*model.User, error)
}

type MeService struct {
	userRepo MeUserRepository
}

func NewMeService(userRepo MeUserRepository) *MeService {
	return &MeService{
		userRepo: userRepo,
	}
}

func (s *MeService) GetProfile(
	ctx context.Context,
	userID uuid.UUID,
) (*dto.MeResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf("find user by ID: %w", err)
	}

	return &dto.MeResponse{
		ID:        user.ID.String(),
		FullName:  user.FullName,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}
