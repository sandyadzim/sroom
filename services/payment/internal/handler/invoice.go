package handler

import (
	"errors"

	"payment/internal/httpapi"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct{ service *service.InvoiceService }

func NewInvoiceHandler(service *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{service: service}
}

func (h *InvoiceHandler) Customer(c *gin.Context) { h.read(c, c.GetString("customer_id")) }
func (h *InvoiceHandler) Staff(c *gin.Context)    { h.read(c, "") }

func (h *InvoiceHandler) read(c *gin.Context, customerID string) {
	id, ok := readPaymentID(c)
	if !ok {
		return
	}
	result, err := h.service.Read(c.Request.Context(), id, customerID, c.Writer.Header().Get("X-Request-ID"))
	if errors.Is(err, service.ErrInvoiceNotAvailable) {
		httpapi.WriteError(c, 409, "INVOICE_NOT_AVAILABLE", "invoice is not available yet")
		return
	}
	if err != nil {
		writeReadPaymentError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
