package router

import (
	"crypto/rand"
	"errors"
	"net/http"
	"strings"

	"payment/internal/handler"
	"payment/internal/httpapi"
	"payment/internal/middleware"

	"github.com/gin-gonic/gin"
)

func New(healthHandler *handler.HealthHandler) (*gin.Engine, error) {
	engine := gin.New()
	if err := engine.SetTrustedProxies(nil); err != nil {
		return nil, errors.New("could not configure trusted proxies")
	}
	engine.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
		if requestID == "" || len(requestID) > 64 || strings.Trim(requestID, allowed) != "" {
			requestID = rand.Text()
		}
		c.Header("X-Request-ID", requestID)
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
		c.Next()
	})
	engine.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		if c.Writer.Written() {
			c.Abort()
			return
		}
		httpapi.WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}))
	engine.NoRoute(func(c *gin.Context) {
		httpapi.WriteError(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})

	health := engine.Group("/health")
	health.GET("/live", healthHandler.Live)
	health.GET("/ready", healthHandler.Ready)
	return engine, nil
}

func RegisterPayment(engine *gin.Engine, verifier middleware.SessionVerifier, paymentHandler *handler.PaymentHandler) {
	api := engine.Group("/api")
	v1 := api.Group("/v1")
	payment := v1.Group("/payment", middleware.RequireCustomer(verifier))
	bookings := payment.Group("/bookings")
	bookings.POST("/:id/payments/va", paymentHandler.CreateVA)
}

func RegisterWebhook(engine *gin.Engine, webhookHandler *handler.WebhookHandler) {
	webhooks := engine.Group("/webhooks")
	webhooks.POST("/midtrans", webhookHandler.Midtrans)
}

func RegisterPaymentReads(engine *gin.Engine, verifier middleware.SessionVerifier, readHandler *handler.ReadPaymentHandler) {
	api := engine.Group("/api")
	v1 := api.Group("/v1")
	payment := v1.Group("/payment")
	bookings := payment.Group("/bookings", middleware.RequireCustomer(verifier))
	bookings.GET("/:id/payment", readHandler.Customer)
	admin := payment.Group("/admin", middleware.RequireStaff(verifier))
	payments := admin.Group("/payments")
	payments.GET("", readHandler.List)
	payments.GET("/:id", readHandler.Detail)
}

func RegisterPaymentRefresh(engine *gin.Engine, verifier middleware.SessionVerifier, refreshHandler *handler.RefreshPaymentHandler) {
	api := engine.Group("/api")
	v1 := api.Group("/v1")
	payment := v1.Group("/payment", middleware.RequireCustomer(verifier))
	bookings := payment.Group("/bookings")
	bookings.POST("/:id/payment/refresh", refreshHandler.Refresh)
}

func RegisterConfirmation(engine *gin.Engine, confirmationHandler *handler.ConfirmationHandler) {
	internal := engine.Group("/internal")
	v1 := internal.Group("/v1")
	bookings := v1.Group("/bookings")
	bookings.GET("/:id/payment-confirmation", confirmationHandler.Get)
	bookings.POST("/:id/expiry-assessment", confirmationHandler.AssessExpiry)
}

func RegisterInvoices(engine *gin.Engine, verifier middleware.SessionVerifier, invoiceHandler *handler.InvoiceHandler) {
	api := engine.Group("/api")
	v1 := api.Group("/v1")
	payments := v1.Group("/payment")
	bookings := payments.Group("/bookings", middleware.RequireCustomer(verifier))
	bookings.GET("/:id/invoice", invoiceHandler.Customer)
	admin := payments.Group("/admin", middleware.RequireStaff(verifier))
	adminBookings := admin.Group("/bookings")
	adminBookings.GET("/:id/invoice", invoiceHandler.Staff)
}
