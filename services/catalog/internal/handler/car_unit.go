package handler

import (
	"log/slog"
	"net/http"
	"net/url"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/response"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
)

type CarUnitHandler struct {
	service *service.CarUnitService
}

func NewCarUnitHandler(unitService *service.CarUnitService) *CarUnitHandler {
	return &CarUnitHandler{service: unitService}
}

func (h *CarUnitHandler) List(c *gin.Context) {
	carID, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid query parameters")
		return
	}
	for field := range query {
		if field != "status" && field != "page" && field != "limit" {
			response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid unit filter")
			return
		}
	}
	page, limit, ok := parsePagination(query)
	status := query.Get("status")
	if !ok || (status != "" && status != model.UnitActive && status != model.UnitInactive) {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid unit status or pagination")
		return
	}
	result, err := h.service.List(c.Request.Context(), carID, status, page, limit)
	if err != nil {
		writeCarUnitError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CarUnitHandler) Create(c *gin.Context) {
	carID, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.CreateCarUnitRequest
	if !readNonNullJSON(c, &req, "unit") {
		return
	}
	unit, err := h.service.Create(c.Request.Context(), carID, req)
	if err != nil {
		writeCarUnitError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Car unit created successfully", "data": unit})
}

func (h *CarUnitHandler) Detail(c *gin.Context) {
	id, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	unit, err := h.service.Detail(c.Request.Context(), id)
	if err != nil {
		writeCarUnitError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": unit})
}

func (h *CarUnitHandler) Update(c *gin.Context) {
	id, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCarUnitRequest
	if !readNonNullJSON(c, &req, "unit") {
		return
	}
	unit, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		writeCarUnitError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Car unit updated successfully", "data": unit})
}

func writeCarUnitError(c *gin.Context, err error) {
	switch err {
	case service.ErrInvalidCarUnitCode, service.ErrInvalidCarUnitStatus, service.ErrEmptyUpdate:
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case repository.ErrCarNotFound:
		response.Error(c, http.StatusNotFound, "CAR_NOT_FOUND", err.Error())
	case repository.ErrCarUnitNotFound:
		response.Error(c, http.StatusNotFound, "CAR_UNIT_NOT_FOUND", err.Error())
	case repository.ErrDemoHoldsActive:
		response.Error(c, http.StatusConflict, "DEMO_HOLDS_ACTIVE", err.Error())
	case repository.ErrCarUnitCodeTaken:
		response.Error(c, http.StatusConflict, "CAR_UNIT_CODE_TAKEN", err.Error())
	default:
		slog.Error("car unit request failed", "service", "catalog", "operation", "car_unit.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
