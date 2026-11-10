package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"identity/internal/identity/response"
)

func requestTestApp(output io.Writer) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: response.ErrorHandler, BodyLimit: MaxBodyBytes})
	UseRequests(app, DefaultRequestTimeout, slog.New(slog.NewJSONHandler(output, nil)).With("service", "identity"))
	return app
}

func TestRequestIDs(t *testing.T) {
	for _, tc := range []struct {
		id    string
		valid bool
	}{
		{"request_123-ABC", true},
		{strings.Repeat("a", 64), true},
		{"", false},
		{strings.Repeat("a", 65), false},
		{"bad id", false},
		{"bad,id", false},
		{"synthetic-private@example.invalid", false},
		{"请求", false},
	} {
		var logs bytes.Buffer
		app := requestTestApp(&logs)
		app.Get("/", func(c fiber.Ctx) error { return errors.New("synthetic-private-error") })
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", tc.id)
		req.Header.Set("Authorization", "Bearer synthetic-private-token")
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		var result response.ErrorResponse
		err = json.NewDecoder(res.Body).Decode(&result)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		id := res.Header.Get("X-Request-ID")
		if tc.valid {
			if id != tc.id {
				t.Fatal("valid request ID must be preserved")
			}
		} else if _, err := uuid.Parse(id); err != nil || id == tc.id {
			t.Fatal("invalid or missing request ID must be replaced by a UUID")
		}
		if result.RequestID != id || result.Error.Code != "INTERNAL_ERROR" {
			t.Fatal("error envelope must echo the validated response ID")
		}
		var event map[string]any
		if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
			t.Fatal("expected a structured error log")
		}
		if event["request_id"] != id || event["operation"] != "http.request" {
			t.Fatal("request ID must correlate the error log and response")
		}
		if strings.Contains(logs.String(), "synthetic-private") {
			t.Fatal("logs must not contain raw error text, rejected header text, or tokens")
		}
	}
}

func TestRequestContextCanceledAfterSuccess(t *testing.T) {
	app := requestTestApp(io.Discard)
	var received context.Context
	app.Get("/", func(c fiber.Ctx) error {
		received = c.Context()
		if _, ok := received.Deadline(); !ok {
			t.Error("handler context must have a deadline")
		}
		return c.SendStatus(204)
	})
	res, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 204 || received == nil || !errors.Is(received.Err(), context.Canceled) {
		t.Fatal("request context must be canceled when the handler finishes")
	}
}

func TestPanicRecovery(t *testing.T) {
	for _, value := range []any{"synthetic-private-panic", errors.New("synthetic-private-panic"), fiber.NewError(400, "synthetic-private-panic")} {
		var logs bytes.Buffer
		app := requestTestApp(&logs)
		var received context.Context
		app.Get("/panic", func(c fiber.Ctx) error {
			received = c.Context()
			panic(value)
		})
		app.Get("/healthy", func(c fiber.Ctx) error { return c.SendStatus(204) })
		res, err := app.Test(httptest.NewRequest("GET", "/panic", nil))
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		var result response.ErrorResponse
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal("panic response must be JSON")
		}
		if res.StatusCode != 500 || result.Error.Code != "INTERNAL_ERROR" || result.RequestID != res.Header.Get("X-Request-ID") {
			t.Fatal("panic must return HTTP 500 with the request ID")
		}
		if strings.Contains(string(body), "synthetic-private") || strings.Contains(logs.String(), "synthetic-private") {
			t.Fatal("panic values must not reach responses or logs")
		}
		if received == nil || !errors.Is(received.Err(), context.Canceled) {
			t.Fatal("panic path must also cancel the request context")
		}
		if res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Frame-Options") != "DENY" {
			t.Fatal("panic path must retain private cache and security headers")
		}
		res, err = app.Test(httptest.NewRequest("GET", "/healthy", nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 204 {
			t.Fatal("server must continue serving after a panic")
		}
	}
}

func TestHeadersOnRateLimitAndNotFound(t *testing.T) {
	app := requestTestApp(io.Discard)
	app.Post("/login", LoginRateLimiter(), func(c fiber.Ctx) error { return c.SendStatus(204) })
	for i := 0; i < 6; i++ {
		res, err := app.Test(httptest.NewRequest("POST", "/login", nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Request-ID") == "" {
			t.Fatal("all rate-limited route responses must have request headers")
		}
		if i == 5 && res.StatusCode != 429 {
			t.Fatal("sixth login request must be rate limited")
		}
	}
	res, err := app.Test(httptest.NewRequest("GET", "/not-found", nil))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 404 || res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("X-Request-ID") == "" || res.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("not-found responses must retain request and security headers")
	}
}
