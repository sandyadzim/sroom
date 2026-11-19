package handler

import (
	"context"
	"net/http"
	"time"

	"payment/internal/httpapi"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	pingDatabase func(context.Context) error
}

func NewHealthHandler(pingDatabase func(context.Context) error) *HealthHandler {
	return &HealthHandler{pingDatabase: pingDatabase}
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := h.pingDatabase(ctx); err != nil {
		httpapi.WriteError(c, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is not ready")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
