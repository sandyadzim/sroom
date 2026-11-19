package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"payment/internal/handler"
	"payment/internal/service"
)

func TestReadPaymentAuthorizationAndValidation(t *testing.T) {
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterPaymentReads(app, testSessions{}, handler.NewReadPaymentHandler(service.NewPaymentService(nil, nil, nil, false)))
	const list = "/api/v1/payment/admin/payments"
	const customer = "/api/v1/payment/bookings/00000000-0000-4000-8000-000000000002/payment"
	for _, test := range []struct {
		name, path, token string
		status            int
		code              string
	}{
		{"guest staff", list, "", 401, "UNAUTHENTICATED"},
		{"customer forbidden", list, "customer", 403, "FORBIDDEN"},
		{"expired staff", list, "expired", 401, "UNAUTHENTICATED"},
		{"identity outage", list, "outage", 503, "DEPENDENCY_UNAVAILABLE"},
		{"guest customer", customer, "", 401, "UNAUTHENTICATED"},
		{"staff forbidden", customer, "admin", 403, "FORBIDDEN"},
		{"owner forbidden", customer, "owner", 403, "FORBIDDEN"},
		{"bad customer UUID", strings.Replace(customer, "00000000-0000-4000-8000-000000000002", "invalid", 1), "customer", 400, "INVALID_REQUEST"},
		{"customer query", customer + "?status=paid", "customer", 400, "INVALID_REQUEST"},
		{"bad payment UUID", list + "/bad", "admin", 400, "INVALID_REQUEST"},
		{"owner passed auth", list + "/bad", "owner", 400, "INVALID_REQUEST"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", test.path, nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}
			res := httptest.NewRecorder()
			app.ServeHTTP(res, req)
			if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", res.Code, res.Body.String())
			}
		})
	}
	for _, query := range []string{"page=0", "page=-1", "page=no", "page=", "limit=0", "limit=101", "limit=", "page=9223372036854775807&limit=20", "status=not_started", "reconciliation=other", "unknown=yes", "page=1&page=2", "search=%ZZ", "status=pending;limit=2", "search=" + strings.Repeat("a", 201)} {
		req := httptest.NewRequest("GET", list+"?"+query, nil)
		req.Header.Set("Authorization", "Bearer admin")
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != 400 {
			t.Fatalf("query=%s response=%d %s", query, res.Code, res.Body.String())
		}
	}
}
