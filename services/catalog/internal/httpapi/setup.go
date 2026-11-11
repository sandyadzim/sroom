package httpapi

import (
	"crypto/rand"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Setup(router *gin.Engine) error {
	if err := router.SetTrustedProxies(nil); err != nil {
		return err
	}

	router.Use(func(c *gin.Context) {
		c.Header("X-Request-ID", rand.Text())
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})

	router.Use(gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		slog.Error("request failed", "request_id", c.Writer.Header().Get("X-Request-ID"))

		if c.Writer.Written() {
			c.Abort()
			return
		}

		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}))

	router.Use(limitBody)

	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "NOT_FOUND", "route not found")
	})

	return nil
}

func limitBody(c *gin.Context) {
	const maxBodyBytes = 16 * 1024

	if c.Request.ContentLength > maxBodyBytes {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	c.Next()
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": gin.H{},
		},
		"request_id": c.Writer.Header().Get("X-Request-ID"),
	})
}
