package response

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
)

type SuccessResponse struct {
	Data any `json:"data"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

type ErrorResponse struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}

func Success(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(SuccessResponse{Data: data})
}

func Error(c fiber.Ctx, status int, code, message string, details map[string]string) error {
	if details == nil {
		details = map[string]string{}
	}
	return c.Status(status).JSON(ErrorResponse{
		Error:     ErrorDetail{Code: code, Message: message, Details: details},
		RequestID: "",
	})
}

func ErrorHandler(c fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError

	if fiberErr, ok := errors.AsType[*fiber.Error](err); ok {
		status = fiberErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		status = fiber.StatusGatewayTimeout
	}

	switch status {
	case fiber.StatusBadRequest:
		return Error(c, status, "INVALID_REQUEST", "invalid request", nil)
	case fiber.StatusUnauthorized:
		return Error(c, status, "UNAUTHENTICATED", "authentication required", nil)
	case fiber.StatusForbidden:
		return Error(c, status, "FORBIDDEN", "access denied", nil)
	case fiber.StatusNotFound:
		return Error(c, status, "NOT_FOUND", "route not found", nil)
	case fiber.StatusMethodNotAllowed:
		return Error(c, status, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	case fiber.StatusRequestEntityTooLarge:
		return Error(c, status, "PAYLOAD_TOO_LARGE", "request body is too large", nil)
	case fiber.StatusUnsupportedMediaType:
		return Error(c, status, "UNSUPPORTED_MEDIA_TYPE", "content type must be application/json", nil)
	case fiber.StatusTooManyRequests:
		return Error(c, status, "RATE_LIMITED", "too many requests", nil)
	case fiber.StatusServiceUnavailable:
		return Error(c, status, "SERVICE_UNAVAILABLE", "service unavailable", nil)
	case fiber.StatusGatewayTimeout:
		return Error(c, status, "REQUEST_TIMEOUT", "request timed out", nil)
	default:
		return Error(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}
