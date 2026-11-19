package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"payment/internal/httpapi"
	"payment/internal/identity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequireStaff(verifier SessionVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		auth := strings.Fields(c.GetHeader("Authorization"))
		if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
			httpapi.WriteError(c, 401, "UNAUTHENTICATED", "authentication required")
			return
		}
		profile, err := verifier.Profile(ctx, auth[1], c.Writer.Header().Get("X-Request-ID"))
		if errors.Is(err, identity.ErrUnauthenticated) {
			httpapi.WriteError(c, 401, "UNAUTHENTICATED", "invalid staff session")
			return
		}
		id, idError := uuid.Parse(profile.ID)
		if err != nil || idError != nil || id == uuid.Nil {
			httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not verify staff session")
			return
		}
		if profile.Role != "ADMIN" && profile.Role != "OWNER" {
			httpapi.WriteError(c, 403, "FORBIDDEN", "staff access required")
			return
		}
		c.Next()
	}
}
