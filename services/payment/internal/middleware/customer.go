package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"payment/internal/httpapi"
	"payment/internal/identity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SessionVerifier interface {
	Profile(context.Context, string, string) (identity.Profile, error)
}

func RequireCustomer(verifier SessionVerifier) gin.HandlerFunc {
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
			httpapi.WriteError(c, 401, "UNAUTHENTICATED", "invalid customer session")
			return
		}
		id, idError := uuid.Parse(profile.ID)
		if err != nil || idError != nil || id == uuid.Nil {
			httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not verify customer session")
			return
		}
		if profile.Role != "CUSTOMER" {
			httpapi.WriteError(c, http.StatusForbidden, "FORBIDDEN", "customer access required")
			return
		}
		c.Set("customer_id", id.String())
		c.Set("customer_full_name", profile.FullName)
		c.Set("customer_email", profile.Email)
		c.Next()
	}
}
