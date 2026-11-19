package handler

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"payment/internal/httpapi"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
)

type ConfirmationHandler struct {
	service *service.PaymentService
	token   string
}

func NewConfirmationHandler(service *service.PaymentService, token string) *ConfirmationHandler {
	return &ConfirmationHandler{service: service, token: token}
}

func (h *ConfirmationHandler) Get(c *gin.Context) {
	auth := strings.Fields(c.GetHeader("Authorization"))
	if h.token == "" || len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") || subtle.ConstantTimeCompare([]byte(auth[1]), []byte(h.token)) != 1 {
		c.Header("WWW-Authenticate", "Bearer")
		httpapi.WriteError(c, 401, "UNAUTHENTICATED", "authentication required")
		return
	}
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	result, err := h.service.Confirmation(ctx, id)
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
