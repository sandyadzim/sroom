package handler

import (
	"errors"
	"log/slog"
	"math"
	"net/url"
	"strings"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/httpapi"
	"payment/internal/repository"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReadPaymentHandler struct{ service *service.PaymentService }

func NewReadPaymentHandler(service *service.PaymentService) *ReadPaymentHandler {
	return &ReadPaymentHandler{service: service}
}

func (h *ReadPaymentHandler) Customer(c *gin.Context) {
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	result, err := h.service.CustomerPayment(c.Request.Context(), id, c.GetString("customer_id"), c.Writer.Header().Get("X-Request-ID"))
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

func (h *ReadPaymentHandler) List(c *gin.Context) {
	query := dto.ListPaymentsQuery{Page: 1, Limit: 20}
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "invalid payment filters")
		return
	}
	for key, items := range values {
		if (key != "page" && key != "limit" && key != "search" && key != "status" && key != "reconciliation") || len(items) != 1 {
			httpapi.WriteError(c, 400, "INVALID_REQUEST", "invalid payment filters")
			return
		}
	}
	if c.ShouldBindQuery(&query) != nil || query.Page > math.MaxInt/query.Limit {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "invalid payment filters")
		return
	}
	query.Search = strings.TrimSpace(query.Search)
	result, total, err := h.service.ListPayments(c.Request.Context(), query)
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result, "page": dto.PaymentPage{Number: query.Page, Limit: query.Limit, Total: total}})
}

func (h *ReadPaymentHandler) Detail(c *gin.Context) {
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	result, err := h.service.PaymentDetail(c.Request.Context(), id, c.Writer.Header().Get("X-Request-ID"))
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

func readPaymentID(c *gin.Context) (string, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil || id == uuid.Nil || len(c.Param("id")) != 36 || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send a UUID without query parameters")
		return "", false
	}
	return id.String(), true
}

func writeReadPaymentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, booking.ErrNotFound):
		httpapi.WriteError(c, 404, "BOOKING_NOT_FOUND", "booking not found")
	case errors.Is(err, repository.ErrPaymentNotFound):
		httpapi.WriteError(c, 404, "PAYMENT_NOT_FOUND", "payment not found")
	default:
		slog.Warn("payment read unavailable", "service", "payment", "operation", "payments.read", "request_id", c.Writer.Header().Get("X-Request-ID"))
		httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not read payment")
	}
}
