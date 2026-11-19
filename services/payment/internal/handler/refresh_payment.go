package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"payment/internal/booking"
	"payment/internal/httpapi"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
)

type RefreshPaymentHandler struct {
	service *service.RefreshPaymentService
}

func NewRefreshPaymentHandler(service *service.RefreshPaymentService) *RefreshPaymentHandler {
	return &RefreshPaymentHandler{service: service}
}

func (h *RefreshPaymentHandler) Refresh(c *gin.Context) {
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send an application/json request")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		httpapi.WriteError(c, 413, "PAYLOAD_TOO_LARGE", "request body is too large")
		return
	}
	// Refresh accepts no client payment facts, only an empty JSON object.
	var input struct{}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	trimmed := bytes.TrimSpace(body)
	if err != nil || len(trimmed) == 0 || trimmed[0] != '{' || decoder.Decode(&input) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send an empty JSON object")
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), id, c.GetString("customer_id"), c.Writer.Header().Get("X-Request-ID"))
	if err == nil {
		c.JSON(200, gin.H{"data": result})
		return
	}
	switch {
	case errors.Is(err, booking.ErrNotFound):
		httpapi.WriteError(c, 404, "BOOKING_NOT_FOUND", "booking not found")
	case errors.Is(err, service.ErrNotConfigured):
		httpapi.WriteError(c, 503, "PAYMENT_NOT_CONFIGURED", "payment integration is not configured")
	default:
		slog.Warn("payment refresh unavailable", "service", "payment", "operation", "payments.refresh", "request_id", c.Writer.Header().Get("X-Request-ID"))
		httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not check payment status")
	}
}
