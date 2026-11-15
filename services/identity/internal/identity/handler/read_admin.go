package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type ReadAdminHandler struct {
	adminService *service.ReadAdminService
}

func NewReadAdminHandler(adminService *service.ReadAdminService) *ReadAdminHandler {
	return &ReadAdminHandler{adminService: adminService}
}

func (h *ReadAdminHandler) List(c fiber.Ctx) error {
	page, limit, search, status, err := accountListQuery(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	}

	result, err := h.adminService.List(c.Context(), page, limit, search, status)
	if err != nil {
		return err
	}
	return c.JSON(result)
}

func (h *ReadAdminHandler) Detail(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid admin ID", nil)
	}
	result, err := h.adminService.Detail(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrAdminNotFound) {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "admin not found", nil)
		}
		return err
	}
	return response.Success(c, fiber.StatusOK, result)
}
