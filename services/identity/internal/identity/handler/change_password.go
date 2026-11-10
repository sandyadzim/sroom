package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/dto"
	"identity/internal/identity/middleware"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type ChangePasswordHandler struct {
	passwordService *service.ChangePasswordService
}

func NewChangePasswordHandler(passwordService *service.ChangePasswordService) *ChangePasswordHandler {
	return &ChangePasswordHandler{passwordService: passwordService}
}

func (h *ChangePasswordHandler) Change(c fiber.Ctx) error {
	session, ok := middleware.GetSession(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "authentication required", nil)
	}
	var req dto.ChangePasswordRequest
	if err := strictJSON(c, &req); err != nil {
		return err
	}
	err := h.passwordService.Change(c.Context(), session.UserID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "invalid session", nil)
		case errors.Is(err, service.ErrPasswordChangeForbidden):
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "access denied", nil)
		case errors.Is(err, service.ErrIncorrectPassword):
			return response.Error(c, fiber.StatusBadRequest, "INVALID_CURRENT_PASSWORD", "current password is incorrect", nil)
		case errors.Is(err, dto.ErrPasswordTooLong):
			return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", map[string]string{"new_password": "password must not exceed 72 bytes"})
		}
		if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
			return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatRegisterErrors(err))
		}
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
