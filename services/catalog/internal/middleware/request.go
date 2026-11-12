package middleware

import (
	"crypto/rand"
	"log/slog"
	"net/http"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func RequestHeaders(c *gin.Context) {
	c.Header("X-Request-ID", rand.Text())
	c.Header("X-Content-Type-Options", "nosniff")
	c.Next()
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, _ any) {
		slog.Error("request failed", "request_id", c.Writer.Header().Get("X-Request-ID"))
		if c.Writer.Written() {
			c.Abort()
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	})
}

func LimitBody(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			response.Error(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
