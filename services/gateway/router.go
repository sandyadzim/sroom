package main

import (
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func newRouter(identityURL, catalogURL, bookingURL *url.URL, logger *slog.Logger) *gin.Engine {
	app := gin.New()
	app.RedirectTrailingSlash = false
	app.HandleMethodNotAllowed = true
	app.ForwardedByClientIP = false
	app.Use(requestHeaders)
	app.Use(browserCORS(envOrDefault("WEB_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")))
	app.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		logger.Error("request failed", "operation", "http.request", "request_id", c.Writer.Header().Get("X-Request-ID"))
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}))
	app.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})
	app.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})
	app.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	client := newHTTPClient()
	identity := forwardTo(client, identityURL, "identity", 16*1024, 8*time.Second, logger)
	catalog := forwardTo(client, catalogURL, "catalog", 16*1024, 8*time.Second, logger)
	upload := forwardTo(client, catalogURL, "catalog", 6*1024*1024, 25*time.Second, logger)

	api := app.Group("/api")
	v1 := api.Group("/v1")
	identityRoutes := v1.Group("/identity")
	identityRoutes.POST("/register", identity)
	identityRoutes.POST("/login", identity)
	identityRoutes.GET("/me", identity)
	identityRoutes.POST("/logout", identity)
	identityRoutes.POST("/change-password", identity)
	admins := identityRoutes.Group("/admins")
	admins.GET("", identity)
	admins.POST("", identity)
	admins.GET("/:id", identity)
	admins.PATCH("/:id", identity)

	customers := identityRoutes.Group("/customers")
	customers.GET("", identity)
	customers.GET("/:id", identity)
	customers.PATCH("/:id/status", identity)

	catalogRoutes := v1.Group("/catalog")
	catalogRoutes.GET("/home/banners", catalog)
	catalogRoutes.GET("/home/featured-cars", catalog)
	catalogRoutes.GET("/brands", catalog)
	catalogRoutes.GET("/cars", catalog)
	catalogRoutes.GET("/cars/:slug", catalog)
	catalogRoutes.GET("/cars/:slug/related", catalog)

	registerCatalogAdmin(catalogRoutes.Group("/admin"), catalog, upload)
	registerStaff(v1.Group("/staff"), client, identityURL, identity, catalog, upload, logger)
	customer := v1.Group("/customer")
	registerCustomerSession(customer, client, identityURL, logger)
	booking := forwardTo(client, bookingURL, "booking", 16*1024, 8*time.Second, logger)
	customer.POST("/checkout/quotes", customerProxy("/api/v1/booking/checkout/quotes", true, booking))
	customer.POST("/bookings", customerProxy("/api/v1/booking/bookings", true, booking))
	return app
}

func registerCatalogAdmin(admin *gin.RouterGroup, catalog, upload gin.HandlerFunc) {
	admin.POST("/images", upload)
	admin.GET("/brands", catalog)
	admin.POST("/brands", catalog)
	admin.PATCH("/brands/:id", catalog)
	admin.GET("/banners", catalog)
	admin.GET("/banners/:position", catalog)
	admin.PUT("/banners/:position", catalog)
	admin.DELETE("/banners/:position", catalog)
	admin.GET("/cars", catalog)
	admin.POST("/cars", catalog)
	admin.GET("/cars/:id", catalog)
	admin.PATCH("/cars/:id", catalog)
	admin.GET("/cars/:id/units", catalog)
	admin.POST("/cars/:id/units", catalog)
	admin.GET("/units/:id", catalog)
	admin.PATCH("/units/:id", catalog)
	admin.GET("/featured-cars", catalog)
	admin.PUT("/featured-cars", catalog)
}

func requestHeaders(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	if !validRequestID(requestID) {
		requestID = rand.Text()
	}
	c.Header("X-Request-ID", requestID)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Next()
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 64 {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.Header("Cache-Control", "no-store")
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": gin.H{},
		},
		"request_id": c.Writer.Header().Get("X-Request-ID"),
	})
}
