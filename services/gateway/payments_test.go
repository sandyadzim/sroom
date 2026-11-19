package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomerPaymentProxy(t *testing.T) {
	calls := 0
	payment := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/api/v1/payment/bookings/fixture/payments/va" || r.Header.Get("Authorization") != "Bearer customer-fixture" || r.Header.Get("Idempotency-Key") != "key-fixture" || r.Header.Get("X-Request-ID") != "payment-proxy-test" || r.Header.Get("Cookie") != "" || string(body) != "{}" {
			t.Error("incorrect Payment forwarding")
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"status":"pending"}}`))
	})
	app := newRouter(payment, payment, payment, payment, testLogger())
	const path = "/api/v1/customer/bookings/fixture/payments/va"
	for _, test := range []struct {
		name, origin string
		cookie       bool
		status       int
	}{
		{"customer", "http://localhost:5173", true, 201},
		{"guest", "http://localhost:5173", false, 401},
		{"cross origin", "https://other.invalid", true, 403},
		{"missing origin", "", true, 403},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", path, strings.NewReader("{}"))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", test.origin)
			req.Header.Set("Idempotency-Key", "key-fixture")
			req.Header.Set("X-Request-ID", "payment-proxy-test")
			req.Header.Set("Authorization", "Bearer forged-fixture")
			if test.cookie {
				req.AddCookie(&http.Cookie{Name: customerCookieName, Value: "customer-fixture"})
			}
			res := request(app, req)
			if res.Code != test.status {
				t.Fatalf("got %d %s", res.Code, res.Body.String())
			}
		})
	}
	if calls != 1 {
		t.Fatalf("provider upstream calls=%d", calls)
	}
	for _, path := range []string{"/api/v1/payment/bookings/fixture/payments/va", "/internal/v1/payments/fixture"} {
		res := request(app, httptest.NewRequest("POST", path, nil))
		if res.Code != 404 {
			t.Fatalf("private/direct route exposed: %s", path)
		}
	}
}
