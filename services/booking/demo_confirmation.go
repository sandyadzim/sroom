package main

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
)

type demoConfirmationRequest struct {
	AmountIDR int64  `json:"amount_idr"`
	Currency  string `json:"currency"`
}

func registerDemoConfirmation(router *gin.Engine, service *checkout.Service, token string, enabled bool, now func() time.Time) {
	if !enabled || token == "" {
		return
	}
	internal := router.Group("/internal")
	v1 := internal.Group("/v1")
	v1.POST("/bookings/:id/demo-confirmation", func(c *gin.Context) {
		if !requirePaymentToken(c, token) {
			return
		}
		id := c.Param("id")
		if !validBookingID(id) {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "booking id must be a UUID")
			return
		}
		var payload demoConfirmationRequest
		if !readCheckoutRequest(c, &payload) {
			return
		}
		if payload.AmountIDR <= 0 || payload.Currency != "IDR" {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "amount_idr must be positive and currency must be IDR")
			return
		}
		requestID := c.Writer.Header().Get("X-Request-ID")
		result, err := service.ConfirmDemoPayment(c.Request.Context(), id, payload.AmountIDR, payload.Currency, now, requestID)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, gin.H{"data": result})
		case errors.Is(err, checkout.ErrBookingNotFound):
			writeError(c, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
		case errors.Is(err, checkout.ErrPaymentAmountMismatch):
			writeError(c, http.StatusUnprocessableEntity, "PAYMENT_AMOUNT_MISMATCH", err.Error())
		case errors.Is(err, checkout.ErrHoldExpired):
			writeError(c, http.StatusConflict, "HOLD_EXPIRED", err.Error())
		case errors.Is(err, checkout.ErrBookingNotConfirmable):
			writeError(c, http.StatusConflict, "BOOKING_NOT_CONFIRMABLE", err.Error())
		default:
			slog.Warn("demo confirmation unavailable", "service", "booking", "operation", "bookings.demo_confirm", "request_id", requestID)
			writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not confirm demo booking")
		}
	})
}
