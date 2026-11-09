package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
)

const MaxBodyBytes = 16 * 1024
const DefaultRequestTimeout = 5 * time.Second

func UseRequests(app *fiber.App, timeout time.Duration, logger *slog.Logger) {
	app.Use(helmet.New(helmet.Config{
		XFrameOptions:         "DENY",
		ContentSecurityPolicy: "default-src 'none'; frame-ancestors 'none'",
	}))
	app.Use(requestContext(timeout, logger))
	app.Use(recoverer.New(recoverer.Config{
		PanicHandler: func(_ fiber.Ctx, _ any) error {
			return fiber.ErrInternalServerError
		},
	}))
}

func requestContext(timeout time.Duration, logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set(fiber.HeaderCacheControl, "no-store")

		requestID := c.Get(fiber.HeaderXRequestID)
		if !validRequestID(requestID) {
			id, err := uuid.NewRandom()
			if err != nil {
				return err
			}
			requestID = id.String()
		}
		c.Set(fiber.HeaderXRequestID, requestID)

		ctx, cancel := context.WithTimeout(c.Context(), timeout)
		defer cancel()
		c.SetContext(ctx)

		err := c.Next()
		if err != nil {
			logger.Error("request failed", "operation", "http.request", "request_id", requestID)
		}
		return err
	}
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
