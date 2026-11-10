package response

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestErrorHandler(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		err    error
	}{
		{400, "INVALID_REQUEST", fiber.NewError(400, "synthetic-private")},
		{401, "UNAUTHENTICATED", fiber.NewError(401, "synthetic-private")},
		{403, "FORBIDDEN", fiber.NewError(403, "synthetic-private")},
		{404, "NOT_FOUND", fiber.NewError(404, "synthetic-private")},
		{405, "METHOD_NOT_ALLOWED", fiber.NewError(405, "synthetic-private")},
		{413, "PAYLOAD_TOO_LARGE", fiber.NewError(413, "synthetic-private")},
		{415, "UNSUPPORTED_MEDIA_TYPE", fiber.NewError(415, "synthetic-private")},
		{429, "RATE_LIMITED", fiber.NewError(429, "synthetic-private")},
		{500, "INTERNAL_ERROR", errors.New("synthetic-private")},
		{503, "SERVICE_UNAVAILABLE", fiber.NewError(503, "synthetic-private")},
		{504, "REQUEST_TIMEOUT", context.DeadlineExceeded},
	} {
		app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
		app.Get("/", func(c fiber.Ctx) error { return tc.err })
		res, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		var result ErrorResponse
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal("expected a JSON error response")
		}
		if res.StatusCode != tc.status || result.Error.Code != tc.code {
			t.Errorf("expected HTTP %d with code %s", tc.status, tc.code)
		}
		if strings.Contains(string(body), "synthetic-private") {
			t.Fatal("raw error message must not be exposed")
		}
	}
}
