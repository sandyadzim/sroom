package middleware

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"identity/internal/identity/model"
	"identity/internal/identity/response"
)

type RoleUserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

func RequireOwner(userRepo RoleUserRepository) fiber.Handler {
	return requireRole(userRepo, model.RoleOwner)
}

func RequireStaff(userRepo RoleUserRepository) fiber.Handler {
	return requireRole(userRepo, model.RoleAdmin, model.RoleOwner)
}

func requireRole(userRepo RoleUserRepository, roles ...model.Role) fiber.Handler {
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
			return fmt.Errorf("find session user: %w", err)
		}

		if !slices.Contains(roles, user.Role) {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "access denied", nil)
		}

		return c.Next()
	}
}
