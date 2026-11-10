package handler

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"identity/internal/identity/model"
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
	page, pageErr := strconv.Atoi(c.Query("page", "1"))
	limit, limitErr := strconv.Atoi(c.Query("limit", "20"))
	if pageErr != nil || limitErr != nil || page < 1 || limit < 1 || limit > 100 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "page must be positive and limit must be between 1 and 100", nil)
	}
	// Keep the offset calculation within the supported integer range.
	if page-1 > math.MaxInt/limit {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "page is too large", nil)
	}

	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != model.StatusActive && status != model.StatusInactive {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "status must be active or inactive", nil)
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
