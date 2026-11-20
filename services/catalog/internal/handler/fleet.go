package handler

import (
	"log/slog"
	"net/http"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *CarHandler) FleetSummary(c *gin.Context) {
	if c.Request.URL.RawQuery != "" {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "fleet summary does not accept query parameters")
		return
	}
	data, err := h.service.FleetSummary(c.Request.Context())
	if err != nil {
		slog.Warn("fleet summary unavailable", "service", "catalog", "operation", "fleet.summary", "request_id", c.Writer.Header().Get("X-Request-ID"))
		response.Error(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not read fleet summary")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}
