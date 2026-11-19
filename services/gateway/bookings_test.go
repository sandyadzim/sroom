package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBookingReadRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	booking := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		wantAuth := "Bearer customer-fixture"
		switch r.URL.Path {
		case "/api/v1/booking/bookings", "/api/v1/booking/bookings/fixture":
		case "/api/v1/booking/admin/bookings", "/api/v1/booking/admin/bookings/fixture":
			wantAuth = "Bearer staff-fixture"
		default:
			t.Errorf("unexpected upstream route %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != wantAuth || r.Header.Get("X-Request-ID") != "read-test" ||
			r.Header.Get("Cookie") != "" || r.URL.RawQuery != "limit=5" {
			t.Error("booking read did not preserve the query or forward the correct session")
		}
		w.WriteHeader(http.StatusOK)
	})
	router := newRouter(booking, booking, booking, testLogger())
	privateRequest := httptest.NewRequest(http.MethodGet, "/internal/v1/bookings/fixture/payment-context", nil)
	privateRequest.Header.Set("Authorization", "Bearer payment-fixture")
	assertError(t, request(router, privateRequest), 404, "NOT_FOUND")
	demoRequest := httptest.NewRequest(http.MethodPost, "/internal/v1/bookings/fixture/demo-confirmation", nil)
	demoRequest.Header.Set("Authorization", "Bearer payment-fixture")
	assertError(t, request(router, demoRequest), 404, "NOT_FOUND")
	for _, route := range []struct{ prefix, cookie, value string }{
		{customerPath, customerCookieName, "customer-fixture"},
		{staffPath, staffCookieName, "staff-fixture"},
	} {
		for _, segment := range []string{"/bookings", "/bookings/fixture"} {
			req := httptest.NewRequest(http.MethodGet, route.prefix+segment+"?limit=5", nil)
			req.AddCookie(&http.Cookie{Name: route.cookie, Value: route.value})
			req.Header.Set("Authorization", "Bearer ignored-fixture")
			req.Header.Set("X-Request-ID", "read-test")
			res := request(router, req)
			if res.Code != 200 || res.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("%s returned %d with Cache-Control=%q", req.URL.Path, res.Code, res.Header().Get("Cache-Control"))
			}
			guest := request(router, httptest.NewRequest(http.MethodGet, route.prefix+segment, nil))
			assertError(t, guest, 401, "UNAUTHENTICATED")
		}
	}
}
