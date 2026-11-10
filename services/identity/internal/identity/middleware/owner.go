package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/response"
)

type OwnerUserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

func RequireOwner(userRepo OwnerUserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		session, ok := GetSession(c)
		if !ok {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "authentication required", nil)
		}

		user, err := userRepo.FindByID(c.Context(), session.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "invalid session", nil)
			}
			return fmt.Errorf("find owner user: %w", err)
		}

		if user.Role != model.RoleOwner {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "access denied", nil)
		}

		return c.Next()
	}
}
