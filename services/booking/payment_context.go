package main

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
)

func registerPaymentContext(router *gin.Engine, service *checkout.Service, token string) {
	if token == "" {
		return
	}
	internal := router.Group("/internal")
	v1 := internal.Group("/v1")
	v1.GET("/bookings/:id/payment-context", func(c *gin.Context) {
		if !requirePaymentToken(c, token) {
			return
		}
		id := c.Param("id")
		if !validBookingID(id) {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "booking id must be a UUID")
			return
		}
		result, err := service.PaymentContext(c.Request.Context(), id)
		if err != nil {
			writeBookingReadError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})
}

func requirePaymentToken(c *gin.Context, token string) bool {
	auth := strings.Fields(c.GetHeader("Authorization"))
	if token == "" || len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") ||
		subtle.ConstantTimeCompare([]byte(auth[1]), []byte(token)) != 1 {
		c.Header("WWW-Authenticate", "Bearer")
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return false
	}
	return true
}
