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
	"payment/internal/dto"
	"payment/internal/httpapi"
	"payment/internal/repository"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentHandler struct{ service *service.PaymentService }

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) CreateVA(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	key, keyError := uuid.Parse(c.GetHeader("Idempotency-Key"))
	if err != nil || keyError != nil || id == uuid.Nil || key == uuid.Nil || len(c.Param("id")) != 36 || len(c.GetHeader("Idempotency-Key")) != 36 || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send UUID booking and idempotency IDs without query parameters")
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
	var input dto.CreateVARequest
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err != nil || len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' || decoder.Decode(&input) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send an empty JSON object")
		return
	}
	input.BookingID, input.CustomerID, input.IdempotencyKey = id.String(), c.GetString("customer_id"), key.String()
	result, created, err := h.service.CreateVA(c.Request.Context(), input, c.Writer.Header().Get("X-Request-ID"))
	if err != nil {
		writePaymentError(c, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	if result.Status == "initiating" || result.ReviewRequired {
		status = http.StatusAccepted
		slog.Warn("payment initialization needs checking", "service", "payment", "operation", "payments.create_va", "request_id", c.Writer.Header().Get("X-Request-ID"))
	}
	c.JSON(status, gin.H{"data": result})
}

func writePaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotConfigured):
		httpapi.WriteError(c, 503, "PAYMENT_NOT_CONFIGURED", err.Error())
	case errors.Is(err, booking.ErrNotFound):
		httpapi.WriteError(c, 404, "BOOKING_NOT_FOUND", "booking not found")
	case errors.Is(err, service.ErrNotPayable), errors.Is(err, repository.ErrSnapshotMismatch), errors.Is(err, repository.ErrInitiationClosed):
		httpapi.WriteError(c, 409, "BOOKING_NOT_PAYABLE", "booking is not payable")
	case errors.Is(err, service.ErrHoldExpired):
		httpapi.WriteError(c, 409, "HOLD_EXPIRED", err.Error())
	case errors.Is(err, service.ErrWindowTooShort):
		httpapi.WriteError(c, 409, "PAYMENT_WINDOW_TOO_SHORT", err.Error())
	case errors.Is(err, repository.ErrIdempotencyConflict):
		httpapi.WriteError(c, 409, "IDEMPOTENCY_CONFLICT", err.Error())
	default:
		slog.Warn("payment dependency unavailable", "service", "payment", "operation", "payments.create_va", "request_id", c.Writer.Header().Get("X-Request-ID"))
		httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not prepare payment")
	}
}
