package router

import (
	"crypto/rand"
	"errors"
	"net/http"
	"strings"

	"payment/internal/handler"
	"payment/internal/httpapi"

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
