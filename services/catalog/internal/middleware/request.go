package middleware

import (
	"crypto/rand"
	"log/slog"
	"net/http"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func RequestHeaders(c *gin.Context) {
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
