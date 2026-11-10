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

var ErrEmailAlreadyExists = errors.New("email already registered")

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
}

type RegisterService struct {
	userRepo UserRepository
}

func NewRegisterService(
	userRepo UserRepository,
) *RegisterService {
	return &RegisterService{
		userRepo: userRepo,
	}
}

func (s *RegisterService) Register(
	ctx context.Context,
	req dto.RegisterRequest,
) (*dto.RegisterResponse, error) {
	// 1. Normalize input
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// 2. Validate request
	if err := dto.ValidateRegister(req); err != nil {
		return nil, err
	}

	// 3. Check whether email already exists
	_, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	// 4. Generate UUID v7
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate user ID: %w", err)
	}

	// 5. Hash password using Argon2id
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// 6. Build user model
	user := &model.User{
		ID:           id,
		FullName:     req.FullName,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         model.RoleCustomer,
		Status:       model.StatusActive,
	}

	// 7. Save user to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	// 8. Return safe response
	return &dto.RegisterResponse{
		ID:       user.ID.String(),
		FullName: user.FullName,
		Email:    user.Email,
	}, nil
}
