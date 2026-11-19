package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"payment/internal/handler"
)

func TestPrivateConfirmationAuthorization(t *testing.T) {
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterConfirmation(app, handler.NewConfirmationHandler(nil, "synthetic-pair-token"))
	for _, test := range []struct {
		token, path string
		status      int
	}{
		{"", "00000000-0000-4000-8000-000000000001", 401},
		{"customer-session", "00000000-0000-4000-8000-000000000001", 401},
		{"synthetic-pair-token", "bad", 400},
		{"synthetic-pair-token", "00000000-0000-4000-8000-000000000001?role=OWNER", 400},
	} {
		req := httptest.NewRequest("GET", "/internal/v1/bookings/"+test.path+"/payment-confirmation", nil)
		// Query belongs after the full route, not in its UUID segment.
		if strings.Contains(test.path, "?") {
			req = httptest.NewRequest("GET", "/internal/v1/bookings/00000000-0000-4000-8000-000000000001/payment-confirmation?role=OWNER", nil)
		}
		if test.token != "" {
			req.Header.Set("Authorization", "Bearer "+test.token)
		}
		response := httptest.NewRecorder()
		app.ServeHTTP(response, req)
		if response.Code != test.status {
			t.Fatalf("status=%d %s", response.Code, response.Body.String())
		}
	}
}
