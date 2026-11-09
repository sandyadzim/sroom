package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/middleware"
	"identity/internal/identity/service"
)

type MeHandler struct {
	meService *service.MeService
}

func NewMeHandler(meService *service.MeService) *MeHandler {
	return &MeHandler{
		meService: meService,
	}
}

func (h *MeHandler) Me(c fiber.Ctx) error {
	// 1. Get authenticated session from middleware
	session, ok := middleware.GetSession(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// 2. Get user profile using session's UserID
	result, err := h.meService.GetProfile(
		c.Context(),
		session.UserID,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid session",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	// 3. Return user profile
	return c.Status(fiber.StatusOK).JSON(result)
}
