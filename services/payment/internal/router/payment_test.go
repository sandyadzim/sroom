package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"payment/internal/handler"
	"payment/internal/identity"
	"payment/internal/service"
)

type testSessions struct{}

func (testSessions) Profile(ctx context.Context, token, _ string) (identity.Profile, error) {
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > 12*time.Second {
		return identity.Profile{}, errors.New("missing request deadline")
	}
	if token == "expired" {
		return identity.Profile{}, identity.ErrUnauthenticated
	}
	if token == "outage" {
		return identity.Profile{}, identity.ErrUnavailable
	}
	role := "CUSTOMER"
	if token == "admin" {
		role = "ADMIN"
	}
	if token == "owner" {
		role = "OWNER"
	}
	return identity.Profile{ID: "00000000-0000-4000-8000-000000000001", Role: role}, nil
}

func TestPaymentRouteAuthorizationAndValidation(t *testing.T) {
	engine, err := New(handler.NewHealthHandler(func(context.Context) error { return nil }))
	if err != nil {
		t.Fatal(err)
	}
	RegisterPayment(engine, testSessions{}, handler.NewPaymentHandler(service.NewPaymentService(nil, nil, nil, false)))
	const path = "/api/v1/payment/bookings/00000000-0000-4000-8000-000000000002/payments/va"
	const key = "00000000-0000-4000-8000-000000000003"
	for _, test := range []struct {
		name, token, body, key, path string
		status                       int
		code                         string
	}{
		{"guest", "", "{}", key, path, 401, "UNAUTHENTICATED"},
		{"expired", "expired", "{}", key, path, 401, "UNAUTHENTICATED"},
		{"identity outage", "outage", "{}", key, path, 503, "DEPENDENCY_UNAVAILABLE"},
		{"admin forbidden", "admin", "{}", key, path, 403, "FORBIDDEN"},
		{"owner forbidden", "owner", "{}", key, path, 403, "FORBIDDEN"},
		{"not configured", "customer", "{}", key, path, 503, "PAYMENT_NOT_CONFIGURED"},
		{"unknown amount", "customer", `{"amount_idr":1}`, key, path, 400, "INVALID_REQUEST"},
		{"unknown identity", "customer", `{"customer_id":"forged"}`, key, path, 400, "INVALID_REQUEST"},
		{"no bank selection", "customer", `{"bank":"bni"}`, key, path, 400, "INVALID_REQUEST"},
		{"null object", "customer", "null", key, path, 400, "INVALID_REQUEST"},
		{"two objects", "customer", "{}{}", key, path, 400, "INVALID_REQUEST"},
		{"empty body", "customer", "", key, path, 400, "INVALID_REQUEST"},
		{"invalid key", "customer", "{}", "invalid", path, 400, "INVALID_REQUEST"},
		{"query forbidden", "customer", "{}", key, path + "?total=1", 400, "INVALID_REQUEST"},
		{"malformed query", "customer", "{}", key, path + "?%zz", 400, "INVALID_REQUEST"},
		{"invalid booking", "customer", "{}", key, "/api/v1/payment/bookings/invalid/payments/va", 400, "INVALID_REQUEST"},
		{"oversized body", "customer", strings.Repeat(" ", 16*1024) + "{}", key, path, 413, "PAYLOAD_TOO_LARGE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", test.key)
			req.Header.Set("X-Request-ID", "payment-route-test")
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, req)
			if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("got %d %s", res.Code, res.Body.String())
			}
			if res.Header().Get("Cache-Control") != "no-store" || !strings.Contains(res.Body.String(), `"request_id":"payment-route-test"`) {
				t.Fatal("missing no-store/request ID")
			}
		})
	}
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if res.Code != 200 {
		t.Fatal("health must work without integration credentials")
	}
}
