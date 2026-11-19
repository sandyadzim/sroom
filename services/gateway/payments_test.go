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

func TestPaymentReadProxy(t *testing.T) {
	calls := 0
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		expectedToken := "staff-fixture"
		expectedPath := "/api/v1/payment/admin/payments"
		switch r.URL.Path {
		case "/api/v1/payment/bookings/fixture/payment":
			expectedToken = "customer-fixture"
			expectedPath = r.URL.Path
		case "/api/v1/payment/admin/payments/fixture":
			expectedPath = r.URL.Path
		}
		if r.Method != "GET" || r.URL.Path != expectedPath || r.Header.Get("Authorization") != "Bearer "+expectedToken || r.Header.Get("Cookie") != "" || r.Header.Get("X-Request-ID") != "read-payment-test" {
			t.Error("incorrect payment read forwarding")
		}
		if r.URL.Path == "/api/v1/payment/admin/payments" && r.URL.RawQuery != "page=2&limit=20&status=pending&reconciliation=clear" {
			t.Error("filters lost")
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	app := newRouter(upstream, upstream, upstream, upstream, testLogger())
	for _, test := range []struct{ path, cookie, value string }{
		{"/api/v1/customer/bookings/fixture/payment", customerCookieName, "customer-fixture"},
		{"/api/v1/staff/payments?page=2&limit=20&status=pending&reconciliation=clear", staffCookieName, "staff-fixture"},
		{"/api/v1/staff/payments/fixture", staffCookieName, "staff-fixture"},
	} {
		req := httptest.NewRequest("GET", test.path, nil)
		req.Header.Set("X-Request-ID", "read-payment-test")
		req.Header.Set("Authorization", "Bearer forged")
		req.AddCookie(&http.Cookie{Name: test.cookie, Value: test.value})
		res := request(app, req)
		if res.Code != 200 || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("response=%d %s", res.Code, res.Body.String())
		}
		guest := request(app, httptest.NewRequest("GET", test.path, nil))
		if guest.Code != 401 {
			t.Fatalf("guest=%d", guest.Code)
		}
		wrongSession := httptest.NewRequest("GET", test.path, nil)
		wrongCookie := staffCookieName
		if test.cookie == staffCookieName {
			wrongCookie = customerCookieName
		}
		wrongSession.AddCookie(&http.Cookie{Name: wrongCookie, Value: "wrong-session"})
		if res := request(app, wrongSession); res.Code != 401 {
			t.Fatalf("wrong cookie=%d", res.Code)
		}
	}
	if calls != 3 {
		t.Fatalf("upstream calls=%d", calls)
	}
	for _, path := range []string{"/api/v1/payment/admin/payments", "/api/v1/payment/bookings/fixture/payment"} {
		if res := request(app, httptest.NewRequest("GET", path, nil)); res.Code != 404 {
			t.Fatalf("direct service path exposed=%d", res.Code)
		}
	}
}

func TestCustomerPaymentRefreshProxy(t *testing.T) {
	calls := 0
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "{}" || r.Method != "POST" || r.URL.Path != "/api/v1/payment/bookings/fixture/payment/refresh" || r.Header.Get("Authorization") != "Bearer customer-fixture" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Request-ID") != "refresh-test" {
			t.Error("incorrect refresh forwarding")
		}
		_, _ = w.Write([]byte(`{"data":{"status":"pending"}}`))
	})
	app := newRouter(upstream, upstream, upstream, upstream, testLogger())
	for _, test := range []struct {
		origin string
		cookie bool
		status int
	}{
		{"http://localhost:5173", true, 200}, {"http://localhost:5173", false, 401}, {"", true, 403}, {"https://other.invalid", true, 403},
	} {
		req := httptest.NewRequest("POST", "/api/v1/customer/bookings/fixture/payment/refresh", strings.NewReader("{}"))
		req.Header.Set("Origin", test.origin)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", "refresh-test")
		req.Header.Set("Authorization", "Bearer forged")
		if test.cookie {
			req.AddCookie(&http.Cookie{Name: customerCookieName, Value: "customer-fixture"})
		}
		res := request(app, req)
		if res.Code != test.status {
			t.Fatalf("refresh=%d %s", res.Code, res.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d", calls)
	}
	if res := request(app, httptest.NewRequest("POST", "/api/v1/payment/bookings/fixture/payment/refresh", strings.NewReader("{}"))); res.Code != 404 {
		t.Fatal("direct Payment route exposed")
	}
}
