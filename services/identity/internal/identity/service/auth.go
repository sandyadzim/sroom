package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/token"
)

var ErrInvalidSession = errors.New("invalid or expired session")

type AuthSessionRepository interface {
	FindByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (*model.Session, error)
}

type AuthService struct {
	sessionRepo AuthSessionRepository
}

func NewAuthService(
	sessionRepo AuthSessionRepository,
) *AuthService {
	return &AuthService{
		sessionRepo: sessionRepo,
	}
}

func (s *AuthService) Authenticate(
	ctx context.Context,
	plainToken string,
) (*model.Session, error) {
	// 1. Reject empty token
	if plainToken == "" {
		return nil, ErrInvalidSession
	}

	// 2. Hash the token
	tokenHash := token.Hash(plainToken)

	// 3. Find active session
	session, err := s.sessionRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidSession
		}

		return nil, fmt.Errorf("find session: %w", err)
	}

	// 4. Return authenticated session
	return session, nil
}
