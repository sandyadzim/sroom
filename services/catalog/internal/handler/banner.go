package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"catalog/internal/dto"
	"catalog/internal/repository"
	"catalog/internal/response"
	"catalog/internal/service"

	"github.com/gin-gonic/gin"
)

type BannerHandler struct {
	service *service.BannerService
}

func NewBannerHandler(bannerService *service.BannerService) *BannerHandler {
	return &BannerHandler{service: bannerService}
}

func (h *BannerHandler) List(c *gin.Context) {
	data, err := h.service.List(c.Request.Context())
	if err != nil {
		writeBannerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *BannerHandler) Detail(c *gin.Context) {
	position, ok := readBannerPosition(c)
	if !ok {
		return
	}
	result, err := h.service.Detail(c.Request.Context(), position)
	if err != nil {
		writeBannerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *BannerHandler) Update(c *gin.Context) {
	position, ok := readBannerPosition(c)
	if !ok {
		return
	}
	var req dto.UpdateBannerRequest
	if !readJSON(c, &req) {
		return
	}
	result, err := h.service.Update(c.Request.Context(), position, req)
	if err != nil {
		writeBannerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Banner saved successfully", "data": result})
}

func (h *BannerHandler) Delete(c *gin.Context) {
	position, ok := readBannerPosition(c)
	if !ok {
		return
	}
	result, err := h.service.Delete(c.Request.Context(), position)
	if err != nil {
		writeBannerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Banner removed successfully", "data": result})
}

func (h *BannerHandler) ListPublic(c *gin.Context) {
	data, err := h.service.ListPublic(c.Request.Context())
	if err != nil {
		writeBannerError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func readBannerPosition(c *gin.Context) (int, bool) {
	position, err := strconv.Atoi(c.Param("position"))
	if err != nil || position < 1 || position > 3 {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "position must be between 1 and 3")
		return 0, false
	}
	return position, true
}

func writeBannerError(c *gin.Context, err error) {
	switch err {
	case service.ErrInvalidBanner, service.ErrInvalidImage:
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case repository.ErrCarNotFound:
		response.Error(c, http.StatusNotFound, "CAR_NOT_FOUND", err.Error())
	case repository.ErrBannerBrand:
		response.Error(c, http.StatusBadRequest, "BRAND_NOT_FOUND", err.Error())
	case repository.ErrBannerNotFound:
		response.Error(c, http.StatusNotFound, "BANNER_NOT_FOUND", err.Error())
	case service.ErrImageUnavailable:
		response.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", err.Error())
	default:
		slog.Error("banner request failed", "service", "catalog", "operation", "banner.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
