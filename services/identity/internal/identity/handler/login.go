package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/dto"
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

	// 1. Parse JSON request
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// 2. Call login service
	result, err := h.loginService.Login(c.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid email or password",
			})

		default:
			if validationErrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error":   "validation failed",
					"details": dto.FormatLoginErrors(validationErrs),
				})
			}

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		}
	}

	// 3. Return successful response
	return c.Status(fiber.StatusOK).JSON(result)
}
