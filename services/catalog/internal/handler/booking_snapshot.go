package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *CarHandler) BookingSnapshot(c *gin.Context) {
	id, ok := readUUIDParam(c, "id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	snapshot, err := h.service.BookingSnapshot(ctx, id)
	if err != nil {
		writeCarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": snapshot})
}
