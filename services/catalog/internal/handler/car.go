package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/response"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CarHandler struct {
	service *service.CarService
}

func NewCarHandler(carService *service.CarService) *CarHandler {
	return &CarHandler{service: carService}
}

func (h *CarHandler) List(c *gin.Context) {
	filters, ok := readCarFilters(c, true)
	if !ok {
		return
	}
	result, err := h.service.List(c.Request.Context(), filters)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CarHandler) Detail(c *gin.Context) {
	id, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	result, err := h.service.Detail(c.Request.Context(), id)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *CarHandler) Create(c *gin.Context) {
	var req dto.CreateCarRequest
	if !readNonNullJSON(c, &req, "car") {
		return
	}
	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Car created successfully", "data": result})
}

func (h *CarHandler) Update(c *gin.Context) {
	id, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCarRequest
	if !readNonNullJSON(c, &req, "car") {
		return
	}
	result, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Car updated successfully", "data": result})
}

func (h *CarHandler) ListPublic(c *gin.Context) {
	filters, ok := readCarFilters(c, false)
	if !ok {
		return
	}
	result, err := h.service.ListPublic(c.Request.Context(), filters)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *CarHandler) DetailPublic(c *gin.Context) {
	result, err := h.service.DetailPublic(c.Request.Context(), c.Param("slug"))
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *CarHandler) ListRelated(c *gin.Context) {
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || (len(query) > 0 && (len(query) != 1 || !query.Has("limit"))) {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid query parameters")
		return
	}
	limit := 3
	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 12 {
			response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "limit must be between 1 and 12")
			return
		}
	}
	data, err := h.service.ListRelated(c.Request.Context(), c.Param("slug"), limit)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *CarHandler) ListPublicBrands(c *gin.Context) {
	status, hasStatus := c.GetQuery("status")
	if hasStatus && status != model.BrandActive {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "status must be active")
		return
	}
	data, err := h.service.ListPublicBrands(c.Request.Context(), hasStatus)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func readUUIDParam(c *gin.Context, param string) (string, bool) {
	id := c.Param(param)
	if len(id) != 36 || uuid.Validate(id) != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", param+" must be a UUID")
		return "", false
	}
	return id, true
}

func readNonNullJSON(c *gin.Context, target any, resource string) bool {
	var raw json.RawMessage
	if !readJSON(c, &raw) {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "send a JSON object")
		return false
	}
	for _, value := range fields {
		if bytes.Equal(value, []byte("null")) {
			response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", resource+" fields must not be null")
			return false
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid "+resource+" JSON")
		return false
	}
	return true
}

func writeCarError(c *gin.Context, err error) {
	switch err {
	case service.ErrInvalidCar, service.ErrInvalidCarImage, service.ErrEmptyUpdate, service.ErrInvalidImage:
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case repository.ErrBrandNotFound:
		response.Error(c, http.StatusBadRequest, "BRAND_NOT_FOUND", err.Error())
	case repository.ErrCarBrandInactive:
		response.Error(c, http.StatusBadRequest, "BRAND_INACTIVE", err.Error())
	case repository.ErrCarNotFound:
		response.Error(c, http.StatusNotFound, "CAR_NOT_FOUND", err.Error())
	case repository.ErrCarAlreadyExists:
		response.Error(c, http.StatusConflict, "CAR_ALREADY_EXISTS", err.Error())
	case repository.ErrCarCoverRequired:
		response.Error(c, http.StatusConflict, "CAR_COVER_REQUIRED", err.Error())
	case service.ErrImageUnavailable:
		response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	default:
		slog.Error("car request failed", "service", "catalog", "operation", "car.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
