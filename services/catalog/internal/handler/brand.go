package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"catalog/internal/dto"
	"catalog/internal/model"
	"catalog/internal/repository"
	"catalog/internal/response"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BrandHandler struct {
	service *service.BrandService
}

func NewBrandHandler(brandService *service.BrandService) *BrandHandler {
	return &BrandHandler{service: brandService}
}

func (h *BrandHandler) List(c *gin.Context) {
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid query parameters")
		return
	}
	page, pageErr := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, limitErr := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if pageErr != nil || limitErr != nil || page < 1 || limit < 1 || limit > 100 {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid page or limit")
		return
	}
	if page-1 > math.MaxInt/limit {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "page is too large")
		return
	}

	search := strings.TrimSpace(query.Get("search"))
	if strings.ContainsRune(search, 0) {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid search")
		return
	}
	result, err := h.service.List(c.Request.Context(), search, page, limit)
	if err != nil {
		writeBrandError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *BrandHandler) Create(c *gin.Context) {
	var req dto.CreateBrandRequest
	if !readBrandJSON(c, &req) {
		return
	}

	result, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		writeBrandError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "Brand created successfully", "data": result})
}

func (h *BrandHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if len(id) != 36 || uuid.Validate(id) != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "id must be a UUID")
		return
	}
	var input dto.UpdateBrandRequest
	if !readBrandJSON(c, &input) {
		return
	}

	var changes model.BrandChanges
	if input.Name != nil {
		if err := json.Unmarshal(input.Name, &changes.Name); err != nil || changes.Name == nil {
			writeBrandError(c, service.ErrInvalidBrandName)
			return
		}
	}

	changes.SetImage = input.ImageObjectKey != nil
	if changes.SetImage {
		if err := json.Unmarshal(input.ImageObjectKey, &changes.ImageObjectKey); err != nil {
			writeBrandError(c, service.ErrInvalidImage)
			return
		}
	}
	if input.Status != nil {
		if err := json.Unmarshal(input.Status, &changes.Status); err != nil || changes.Status == nil {
			writeBrandError(c, service.ErrInvalidBrandStatus)
			return
		}
	}

	result, err := h.service.Update(c.Request.Context(), id, changes)
	if err != nil {
		writeBrandError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Brand updated successfully", "data": result})
}

func readBrandJSON(c *gin.Context, target any) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		response.Error(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json")
		return false
	}

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(target)
	if err == nil {
		// The body must contain exactly one JSON object.
		err = decoder.Decode(new(json.RawMessage))
		if err == io.EOF {
			return true
		}
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return false
	}
	response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "send a valid JSON object")
	return false
}

func writeBrandError(c *gin.Context, err error) {
	switch err {
	case service.ErrInvalidBrandName, service.ErrInvalidBrandStatus, service.ErrEmptyUpdate, service.ErrInvalidImage:
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case repository.ErrBrandNotFound:
		response.Error(c, http.StatusNotFound, "BRAND_NOT_FOUND", err.Error())
	case repository.ErrBrandNameTaken:
		response.Error(c, http.StatusConflict, "BRAND_NAME_TAKEN", err.Error())
	case service.ErrImageUnavailable:
		response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	default:
		slog.Error("brand request failed", "service", "catalog", "operation", "brand.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
