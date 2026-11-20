package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"payment/internal/httpapi"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
)

func (h *ConfirmationHandler) AssessExpiry(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		httpapi.WriteError(c, 413, "PAYLOAD_TOO_LARGE", "request body is too large")
		return
	}
	if err != nil || len(body) != 0 {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send this request without a body")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	result, err := h.service.AssessExpiry(ctx, id, c.Writer.Header().Get("X-Request-ID"))
	if errors.Is(err, service.ErrNotExpirable) {
		httpapi.WriteError(c, 409, "BOOKING_NOT_EXPIRABLE", "booking is not pending or its deadline has not been reached")
		return
	}
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
