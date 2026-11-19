package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"payment/internal/handler"
)

func TestRefreshAuthorizationAndBody(t *testing.T) {
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterPaymentRefresh(app, testSessions{}, handler.NewRefreshPaymentHandler(nil))
	const path = "/api/v1/payment/bookings/00000000-0000-4000-8000-000000000002/payment/refresh"
	for _, test := range []struct {
		name, token, body, contentType, path string
		status                               int
		code                                 string
	}{
		{"guest", "", "{}", "application/json", path, 401, "UNAUTHENTICATED"},
		{"expired", "expired", "{}", "application/json", path, 401, "UNAUTHENTICATED"},
		{"admin", "admin", "{}", "application/json", path, 403, "FORBIDDEN"},
		{"owner", "owner", "{}", "application/json", path, 403, "FORBIDDEN"},
		{"identity unavailable", "outage", "{}", "application/json", path, 503, "DEPENDENCY_UNAVAILABLE"},
		{"invalid UUID", "customer", "{}", "application/json", strings.Replace(path, "00000000-0000-4000-8000-000000000002", "bad", 1), 400, "INVALID_REQUEST"},
		{"query", "customer", "{}", "application/json", path + "?status=paid", 400, "INVALID_REQUEST"},
		{"unknown status claim", "customer", `{"status":"paid"}`, "application/json", path, 400, "INVALID_REQUEST"},
		{"unknown amount", "customer", `{"amount_idr":1}`, "application/json", path, 400, "INVALID_REQUEST"},
		{"null", "customer", "null", "application/json", path, 400, "INVALID_REQUEST"},
		{"array", "customer", "[]", "application/json", path, 400, "INVALID_REQUEST"},
		{"multiple objects", "customer", "{}{}", "application/json", path, 400, "INVALID_REQUEST"},
		{"empty", "customer", "", "application/json", path, 400, "INVALID_REQUEST"},
		{"content type", "customer", "{}", "text/plain", path, 400, "INVALID_REQUEST"},
		{"oversized", "customer", strings.Repeat("a", 16385), "application/json", path, 413, "PAYLOAD_TOO_LARGE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", test.path, strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.contentType)
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
}
