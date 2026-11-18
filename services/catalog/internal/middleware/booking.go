package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"catalog/internal/response"

	"github.com/gin-gonic/gin"
)

func RequireBookingToken(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		auth := strings.Fields(c.GetHeader("Authorization"))
		if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") ||
			subtle.ConstantTimeCompare([]byte(auth[1]), []byte(token)) != 1 {
			c.Header("WWW-Authenticate", "Bearer")
			response.Error(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}
		c.Next()
	}
}
