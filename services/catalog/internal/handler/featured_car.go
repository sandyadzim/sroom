package handler

import (
	"log/slog"
	"net/http"

	"catalog/internal/dto"
	"catalog/internal/repository"
	"catalog/internal/response"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
)

type FeaturedCarHandler struct {
	service *service.FeaturedCarService
}

func NewFeaturedCarHandler(featuredCarService *service.FeaturedCarService) *FeaturedCarHandler {
	return &FeaturedCarHandler{service: featuredCarService}
}

func (h *FeaturedCarHandler) List(c *gin.Context) {
	data, err := h.service.List(c.Request.Context())
	if err != nil {
		writeFeaturedCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *FeaturedCarHandler) Replace(c *gin.Context) {
	var req dto.UpdateFeaturedCarsRequest
	if !readJSON(c, &req) {
		return
	}
	data, err := h.service.Replace(c.Request.Context(), req)
	if err != nil {
		writeFeaturedCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Featured cars saved successfully", "data": data})
}

func (h *FeaturedCarHandler) ListPublic(c *gin.Context) {
	data, err := h.service.ListPublic(c.Request.Context())
	if err != nil {
		writeFeaturedCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func writeFeaturedCarError(c *gin.Context, err error) {
	switch err {
	case service.ErrInvalidFeaturedCars:
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case repository.ErrCarNotFound:
		response.Error(c, http.StatusNotFound, "CAR_NOT_FOUND", err.Error())
	default:
		slog.Error("featured cars request failed", "service", "catalog", "operation", "featured_car.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
