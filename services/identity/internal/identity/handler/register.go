package handler

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/dto"
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

    if err := c.Bind().Body(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "invalid request body",
        })
    }

    result, err := h.registerService.Register(c.Context(), req)
    if err != nil {
        switch {
        case errors.Is(err, service.ErrEmailAlreadyExists):
            return c.Status(fiber.StatusConflict).JSON(fiber.Map{
                "error": "email already registered",
            })

        case errors.Is(err, dto.ErrPasswordTooLong):
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error":   "validation failed",
                "details": dto.FormatRegisterErrors(err),
            })

        default:
            if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
                return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                    "error":   "validation failed",
                    "details": dto.FormatRegisterErrors(err),
                })
            }

            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "internal server error",
            })
        }
    }


    return c.Status(fiber.StatusCreated).JSON(result)
}
