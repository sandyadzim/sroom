package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/model"
	"identity/internal/identity/response"
	"identity/internal/identity/service"
)

const SessionContextKey = "auth_session"

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(
	authService *service.AuthService,
) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (m *AuthMiddleware) RequireAuth(c fiber.Ctx) error {
	authorization := c.Get(fiber.HeaderAuthorization)

	// 1. Check Authorization header format
	parts := strings.Fields(authorization)

	if len(parts) != 2 ||
		!strings.EqualFold(parts[0], "Bearer") {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "missing or invalid authorization header", nil)
	}

	plainToken := parts[1]

	// 2. Authenticate token
	session, err := m.authService.Authenticate(
		c.Context(),
		plainToken,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSession) {
			return response.Error(c, fiber.StatusUnauthorized, "UNAUTHENTICATED", "invalid or expired session", nil)
		}

		return err
	}

	// 3. Store authenticated session in request context
	c.Locals(SessionContextKey, session)

	// 4. Continue to the next handler
	return c.Next()
}

func GetSession(c fiber.Ctx) (*model.Session, bool) {
	session, ok := c.Locals(SessionContextKey).(*model.Session)

	if !ok || session == nil {
		return nil, false
	}

	return session, true
}
