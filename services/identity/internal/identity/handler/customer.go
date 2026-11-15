package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"identity/internal/identity/dto"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type CustomerHandler struct {
	service *service.CustomerService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: customerService}
}

func (h *CustomerHandler) List(c fiber.Ctx) error {
	page, limit, search, status, err := accountListQuery(c)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	}

	customers, err := h.service.List(c.Context(), page, limit, search, status)
	if err != nil {
		return err
	}
	return c.JSON(customers)
}

func (h *CustomerHandler) Detail(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid customer ID", nil)
	}
	customer, err := h.service.Detail(c.Context(), id)
	if err != nil {
		return customerError(c, err)
	}
	return response.Success(c, fiber.StatusOK, customer)
}

func (h *CustomerHandler) UpdateStatus(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid customer ID", nil)
	}
	var req dto.UpdateCustomerStatusRequest
	if err := strictJSON(c, &req); err != nil {
		return err
	}
	customer, err := h.service.UpdateStatus(c.Context(), id, req)
	if err != nil {
		return customerError(c, err)
	}
	return response.Success(c, fiber.StatusOK, customer)
}

func customerError(c fiber.Ctx, err error) error {
	if errors.Is(err, service.ErrCustomerNotFound) {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "customer not found", nil)
	}
	if errors.Is(err, service.ErrInvalidCustomerStatus) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", err.Error(), nil)
	}
	return err
}
