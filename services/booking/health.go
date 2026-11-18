package main

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func newRouter(pingDatabase func(context.Context) error) (*gin.Engine, error) {
	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, errors.New("could not configure trusted proxies")
	}
	router.Use(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
		if requestID == "" || len(requestID) > 64 || strings.Trim(requestID, allowed) != "" {
			requestID = rand.Text()
		}
		c.Header("X-Request-ID", requestID)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	router.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		if c.Writer.Written() {
			c.Abort()
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}))
	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})

	health := router.Group("/health")
	health.GET("/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	health.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pingDatabase(ctx); err != nil {
			writeError(c, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is not ready")
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return router, nil
}

func writeError(c *gin.Context, status int, code, message string) {
	writeErrorDetails(c, status, code, message, gin.H{})
}

func writeErrorDetails(c *gin.Context, status int, code, message string, details gin.H) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": details,
		},
		"request_id": c.Writer.Header().Get("X-Request-ID"),
	})
}
