package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/dto"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type LoginHandler struct {
	loginService *service.LoginService
}

func NewLoginHandler(
	loginService *service.LoginService,
) *LoginHandler {
	return &LoginHandler{
		loginService: loginService,
	}
}

func (h *LoginHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if !c.Is("json") {
		return fiber.ErrUnsupportedMediaType
	}

	// 1. Parse JSON request
	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}

	// 2. Call login service
	result, err := h.loginService.Login(c.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return response.Error(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password", nil)

		default:
			if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
				return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatLoginErrors(validationErrs))
			}

			return err
		}
	}

	// 3. Return successful response
	return response.Success(c, fiber.StatusOK, result)
}
