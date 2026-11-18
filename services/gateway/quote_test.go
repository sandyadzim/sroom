package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCustomerQuoteProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("WEB_ORIGINS", "http://localhost:5173")
	const quotePath = "/api/v1/customer/checkout/quotes"
	const body = `{"variant_id":"fixture","pickup_at":"fixture","return_at":"fixture"}`
	var forwarded atomic.Int32
	booking := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		payload, err := io.ReadAll(r.Body)
		if err != nil || string(payload) != body || r.Method != http.MethodPost ||
			r.URL.Path != "/api/v1/booking/checkout/quotes" || r.Header.Get("X-Request-ID") != "quote-test" ||
			r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Cookie") != "" {
			t.Error("unexpected quote request at Booking")
		}
		switch r.Header.Get("Authorization") {
		case "Bearer customer-session":
			_, _ = io.WriteString(w, `{"data":{"available":true}}`)
		case "Bearer staff-session":
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":{"code":"CUSTOMER_REQUIRED"}}`)
		default:
			t.Error("customer cookie was not forwarded as a Bearer token")
		}
	})
	router := newRouter(booking, booking, booking, testLogger())

	for _, test := range []struct {
		name, origin, cookie string
		status               int
		wantForward          bool
	}{
		{"customer", "http://localhost:5173", "customer-session", 200, true},
		{"guest", "http://localhost:5173", "", 401, false},
		{"wrong role from Booking", "http://localhost:5173", "staff-session", 403, true},
		{"wrong origin", "http://invalid.test", "customer-session", 403, false},
		{"missing origin", "", "customer-session", 403, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := forwarded.Load()
			req := httptest.NewRequest(http.MethodPost, quotePath, strings.NewReader(body))
			req.Header.Set("Origin", test.origin)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", "quote-test")
			req.Header.Set("Authorization", "Bearer ignored-header")
			if test.cookie != "" {
				req.AddCookie(&http.Cookie{Name: customerCookieName, Value: test.cookie})
			}
			res := request(router, req)
			if res.Code != test.status || res.Header().Get("X-Request-ID") != "quote-test" ||
				res.Header().Get("Cache-Control") != "no-store" || (forwarded.Load() > before) != test.wantForward {
				t.Fatalf("unexpected quote response: %d %s", res.Code, res.Body.String())
			}
		})
	}
	assertError(t, request(router, httptest.NewRequest(http.MethodPost, "/api/v1/booking/checkout/quotes", nil)), 404, "NOT_FOUND")
}
