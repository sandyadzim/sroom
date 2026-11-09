package handler

import (
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/middleware"
	"identity/internal/identity/service"
)

type LogoutHandler struct {
	logoutService *service.LogoutService
}

func NewLogoutHandler(
	logoutService *service.LogoutService,
) *LogoutHandler {
	return &LogoutHandler{
		logoutService: logoutService,
	}
}

func (h *LogoutHandler) Logout(c fiber.Ctx) error {
	// 1. Get authenticated session from middleware
	session, ok := middleware.GetSession(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// 2. Revoke the current session
	if err := h.logoutService.Logout(
		c.Context(),
		session.ID,
	); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	// 3. Return success without response body
	return c.SendStatus(fiber.StatusNoContent)
}
