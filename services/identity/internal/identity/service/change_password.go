package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
)

var ErrIncorrectPassword = errors.New("current password is incorrect")
var ErrPasswordChangeForbidden = errors.New("password change is only available to OWNER and ADMIN")

type ChangePasswordRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error
}

type ChangePasswordService struct {
	userRepo ChangePasswordRepository
}

func NewChangePasswordService(userRepo ChangePasswordRepository) *ChangePasswordService {
	return &ChangePasswordService{userRepo: userRepo}
}

func (s *ChangePasswordService) Change(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	if err := dto.ValidateChangePassword(req); err != nil {
		return err
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("find password owner: %w", err)
	}
	if user.Role != model.RoleOwner && user.Role != model.RoleAdmin {
		return ErrPasswordChangeForbidden
	}
	valid, err := password.Verify(req.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify current password: %w", err)
	}
	if !valid {
		return ErrIncorrectPassword
	}
	hash, err := password.Hash(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("update own password: %w", err)
	}
	return nil
}
