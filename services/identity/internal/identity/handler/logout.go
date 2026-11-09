package handler

import (
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/middleware"
	"identity/internal/identity/response"
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
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "authentication required", nil)
	}

	// 2. Revoke the current session
	if err := h.logoutService.Logout(
		c.Context(),
		session.ID,
	); err != nil {
		return err
	}

	// 3. Return success without response body
	return c.SendStatus(fiber.StatusNoContent)
}
