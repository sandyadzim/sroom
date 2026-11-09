package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type LogoutSessionRepository interface {
	DeleteByID(
		ctx context.Context,
		sessionID uuid.UUID,
	) error
}

type LogoutService struct {
	sessionRepo LogoutSessionRepository
}

func NewLogoutService(
	sessionRepo LogoutSessionRepository,
) *LogoutService {
	return &LogoutService{
		sessionRepo: sessionRepo,
	}
}

func (s *LogoutService) Logout(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	if err := s.sessionRepo.DeleteByID(ctx, sessionID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
