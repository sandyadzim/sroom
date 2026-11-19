package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"payment/internal/handler"
)

func TestHealth(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		pingError  error
		wantPing   bool
		wantStatus int
		wantCode   string
	}{
		{"live with unavailable database", "/health/live", errors.New("private database details"), false, 200, ""},
		{"ready", "/health/ready", nil, true, 200, ""},
		{"database unavailable", "/health/ready", errors.New("private database details"), true, 503, "DATABASE_UNAVAILABLE"},
		{"unknown route", "/missing", nil, false, 404, "NOT_FOUND"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pinged := false
			engine, err := New(handler.NewHealthHandler(func(ctx context.Context) error {
				pinged = true
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second {
					t.Error("database ping must have a deadline of at most two seconds")
				}
				return test.pingError
			}))
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("X-Request-ID", "payment-health-1")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != test.wantStatus || pinged != test.wantPing {
				t.Fatalf("status = %d, pinged = %t", response.Code, pinged)
			}
			var body struct {
				Status string `json:"status"`
				Error  struct {
					Code    string         `json:"code"`
					Details map[string]any `json:"details"`
				} `json:"error"`
				RequestID string `json:"request_id"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if test.wantCode == "" {
				if body.Status != "ok" {
					t.Fatalf("unexpected health response: %s", response.Body.String())
				}
			} else if body.Error.Code != test.wantCode || body.Error.Details == nil || body.RequestID != "payment-health-1" {
				t.Fatalf("unexpected error envelope: %s", response.Body.String())
			}
			if strings.Contains(response.Body.String(), "private database details") {
				t.Fatal("response exposed database details")
			}
			if response.Header().Get("X-Request-ID") != "payment-health-1" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("request ID or no-store header missing")
			}
		})
	}
}

func TestReadinessUsesRequestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine, err := New(handler.NewHealthHandler(func(ctx context.Context) error {
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("database ping did not inherit request cancellation")
		}
		return ctx.Err()
	}))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil).WithContext(ctx))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestRequestIDAndRecovery(t *testing.T) {
	for _, requestID := range []string{"", "bad request id", strings.Repeat("a", 65)} {
		engine, err := New(handler.NewHealthHandler(func(context.Context) error {
			panic("private panic details")
		}))
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		request.Header.Set("X-Request-ID", requestID)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		generatedID := response.Header().Get("X-Request-ID")
		if generatedID == "" || generatedID == requestID || len(generatedID) > 64 {
			t.Fatal("missing or invalid generated request ID")
		}
		if response.Code != 500 || !strings.Contains(response.Body.String(), `"code":"INTERNAL_ERROR"`) || strings.Contains(response.Body.String(), "private panic details") {
			t.Fatalf("unsafe recovery response: %d %s", response.Code, response.Body.String())
		}
	}
}
