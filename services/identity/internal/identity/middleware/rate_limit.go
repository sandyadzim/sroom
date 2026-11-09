package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func LoginRateLimiter() fiber.Handler {
	return newRateLimiter(5)
}

func RegisterRateLimiter() fiber.Handler {
	return newRateLimiter(3)
}

func newRateLimiter(max int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: time.Minute,

		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP()
		},

		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,

		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(
				fiber.Map{
					"error": "too many requests",
				},
			)
		},
	})
}
