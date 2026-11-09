package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/middleware"
	"identity/internal/identity/response"
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
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "authentication required", nil)
	}

	// 2. Get user profile using session's UserID
	result, err := h.meService.GetProfile(
		c.Context(),
		session.UserID,
	)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "invalid session", nil)
		}

		return err
	}

	// 3. Return user profile
	return response.Success(c, fiber.StatusOK, result)
}
