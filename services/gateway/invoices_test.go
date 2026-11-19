package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvoiceCookieForwarding(t *testing.T) {
	calls := 0
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		wantPath, wantToken := "/api/v1/payment/bookings/fixture/invoice", "customer-fixture"
		if r.URL.Path == "/api/v1/payment/admin/bookings/fixture/invoice" {
			wantPath, wantToken = r.URL.Path, "staff-fixture"
		}
		if r.URL.Path != wantPath || r.Method != "GET" || r.Header.Get("Authorization") != "Bearer "+wantToken || r.Header.Get("Cookie") != "" || r.Header.Get("X-Request-ID") != "invoice-test" {
			t.Error("incorrect invoice forwarding")
		}
		_, _ = w.Write([]byte(`{"data":{"invoice_number":"synthetic"}}`))
	})
	app := newRouter(upstream, upstream, upstream, upstream, testLogger())
	for _, test := range []struct{ path, cookie, token string }{
		{"/api/v1/customer/bookings/fixture/invoice", customerCookieName, "customer-fixture"}, {"/api/v1/staff/bookings/fixture/invoice", staffCookieName, "staff-fixture"},
	} {
		req := httptest.NewRequest("GET", test.path, nil)
		req.AddCookie(&http.Cookie{Name: test.cookie, Value: test.token})
		req.Header.Set("Authorization", "Bearer forged")
		req.Header.Set("X-Request-ID", "invoice-test")
		res := request(app, req)
		if res.Code != 200 {
			t.Fatalf("invoice status=%d", res.Code)
		}
	}
	if calls != 2 {
		t.Fatal("invoice did not reach Payment")
	}
	for _, path := range []string{"/internal/v1/bookings/fixture/invoice-context", "/api/v1/payment/bookings/fixture/invoice"} {
		if res := request(app, httptest.NewRequest("GET", path, nil)); res.Code != 404 {
			t.Fatal("private route exposed")
		}
	}
}
