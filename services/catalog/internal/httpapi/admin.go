package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"catalog/internal/identity"

	"github.com/gin-gonic/gin"
)

func RequireAdmin(client *identity.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		authorization := strings.Fields(c.GetHeader("Authorization"))
		if len(authorization) != 2 || !strings.EqualFold(authorization[0], "Bearer") {
			writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
			return
		}

		role, err := client.Role(c.Request.Context(), authorization[1], c.Writer.Header().Get("X-Request-ID"))
		if errors.Is(err, identity.ErrUnauthenticated) {
			writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid session")
			return
		}
		if err != nil {
			writeError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "identity verification unavailable")
			return
		}
		if role != "OWNER" && role != "ADMIN" {
			writeError(c, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}

		c.Next()
	}
}
