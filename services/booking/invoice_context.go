package main

import (
	"errors"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
)

func registerInvoiceContext(router *gin.Engine, service *checkout.Service, token string) {
	if token == "" {
		return
	}
	internal := router.Group("/internal")
	v1 := internal.Group("/v1")
	bookings := v1.Group("/bookings")
	bookings.GET("/:id/invoice-context", func(c *gin.Context) {
		if !requirePaymentToken(c, token) {
			return
		}
		id := c.Param("id")
		if !validBookingID(id) || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
			writeError(c, 400, "INVALID_REQUEST", "send a UUID without query parameters")
			return
		}
		result, err := service.InvoiceContext(c.Request.Context(), id)
		if errors.Is(err, checkout.ErrInvoiceNotReady) {
			writeError(c, 409, "INVOICE_NOT_AVAILABLE", "booking has no verified confirmation")
			return
		}
		if err != nil {
			writeBookingReadError(c, err)
			return
		}
		c.JSON(200, gin.H{"data": result})
	})
}
