package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"identity/internal/identity/dto"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

type AdminHandler struct {
	createService *service.CreateAdminService
	updateService *service.UpdateAdminService
}

func NewAdminHandler(createService *service.CreateAdminService, updateService *service.UpdateAdminService) *AdminHandler {
	return &AdminHandler{createService: createService, updateService: updateService}
}

func (h *AdminHandler) Create(c fiber.Ctx) error {
	var req dto.CreateAdminRequest
	if err := strictJSON(c, &req); err != nil {
		return err
	}
	result, err := h.createService.Create(c.Context(), req)
	if err != nil {
		return adminWriteError(c, err)
	}
	return response.Success(c, fiber.StatusCreated, result)
}

func (h *AdminHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid admin ID", nil)
	}
	var req dto.UpdateAdminRequest
	if err := strictJSON(c, &req); err != nil {
		return err
	}
	result, err := h.updateService.Update(c.Context(), id, req)
	if err != nil {
		return adminWriteError(c, err)
	}
	return response.Success(c, fiber.StatusOK, result)
}

func strictJSON(c fiber.Ctx, target any) error {
	if !c.Is("json") {
		return fiber.ErrUnsupportedMediaType
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &fields); err != nil || fields == nil {
		return fiber.ErrBadRequest
	}
	for name, value := range fields {
		if name != strings.ToLower(name) || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fiber.ErrBadRequest
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fiber.ErrBadRequest
	}
	return nil
}

func adminWriteError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrEmailAlreadyExists):
		return response.Error(c, fiber.StatusConflict, "EMAIL_ALREADY_REGISTERED", "email already registered", nil)
	case errors.Is(err, service.ErrAdminNotFound):
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "admin not found", nil)
	case errors.Is(err, dto.ErrEmptyUpdate):
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "at least one field is required", nil)
	case errors.Is(err, dto.ErrPasswordTooLong):
		return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatRegisterErrors(err))
	}
	if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return response.Error(c, fiber.StatusBadRequest, "VALIDATION_ERROR", "validation failed", dto.FormatRegisterErrors(err))
	}
	return err
}
