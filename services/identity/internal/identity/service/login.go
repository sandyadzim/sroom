package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/dto"
	"identity/internal/identity/model"
	"identity/internal/identity/password"
	"identity/internal/identity/token"
)

const sessionDuration = 7 * 24 * time.Hour

var ErrInvalidCredentials = errors.New("invalid email or password")

type LoginUserRepository interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*model.User, error)
}

type LoginSessionRepository interface {
	Create(
		ctx context.Context,
		session *model.Session,
	) error
}

type LoginService struct {
	userRepo    LoginUserRepository
	sessionRepo LoginSessionRepository
}

func NewLoginService(
	userRepo LoginUserRepository,
	sessionRepo LoginSessionRepository,
) *LoginService {
	return &LoginService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (s *LoginService) Login(
	ctx context.Context,
	req dto.LoginRequest,
) (*dto.LoginResponse, error) {
	// 1. Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// 2. Validate request
	if err := dto.ValidateLogin(req); err != nil {
		return nil, err
	}

	// 3. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find user by email: %w", err)
	}

	// 4. Verify password using Argon2id
	valid, err := password.Verify(req.Password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}

	if !valid {
		return nil, ErrInvalidCredentials
	}

	// 5. Generate opaque token
	plainToken, err := token.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// 6. Generate UUID v7 for session
	sessionID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate session ID: %w", err)
	}

	// 7. Set session expiration
	expiresAt := time.Now().UTC().Add(sessionDuration)

	// 8. Build session model
	session := &model.Session{
		ID:        sessionID,
		UserID:    user.ID,
		TokenHash: token.Hash(plainToken),
		ExpiresAt: expiresAt,
	}

	// 9. Save session to database
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// 10. Return token to client
	return &dto.LoginResponse{
		Token:     plainToken,
		TokenType: "Bearer",
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}
