package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/dto"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type RegisterHandler struct {
	registerService *service.RegisterService
}

func NewRegisterHandler(
	registerService *service.RegisterService,
) *RegisterHandler {
	return &RegisterHandler{
		registerService: registerService,
	}
}

func (h *RegisterHandler) Register(c fiber.Ctx) error {
	var req dto.RegisterRequest
	if !c.Is("json") {
		return fiber.ErrUnsupportedMediaType
	}

	if err := c.Bind().JSON(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request body", nil)
	}

	result, err := h.registerService.Register(c.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			return response.Error(c, fiber.StatusConflict, "EMAIL_ALREADY_REGISTERED", "email already registered", nil)

		case errors.Is(err, dto.ErrPasswordTooLong):
			return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatRegisterErrors(err))

		default:
			if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
				return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatRegisterErrors(err))
			}

			return err
		}
	}

	return response.Success(c, fiber.StatusCreated, result)
}
