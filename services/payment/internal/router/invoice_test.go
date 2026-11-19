package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"payment/internal/handler"
)

func TestInvoiceAuthorizationAndValidation(t *testing.T) {
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterInvoices(app, testSessions{}, handler.NewInvoiceHandler(nil))
	const customer = "/api/v1/payment/bookings/00000000-0000-4000-8000-000000000001/invoice"
	const staff = "/api/v1/payment/admin/bookings/00000000-0000-4000-8000-000000000001/invoice"
	for _, test := range []struct {
		path, token string
		status      int
	}{
		{customer, "", 401}, {staff, "", 401}, {staff, "customer", 403}, {customer, "admin", 403}, {customer, "owner", 403}, {customer, "expired", 401}, {customer, "outage", 503},
		{customer + "?price=1", "customer", 400}, {staff + "?customer_id=forged", "admin", 400}, {strings.Replace(staff, "00000000-0000-4000-8000-000000000001", "bad", 1), "owner", 400},
	} {
		req := httptest.NewRequest("GET", test.path, nil)
		req.Header.Set("Authorization", "Bearer "+test.token)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != test.status {
			t.Fatalf("status=%d want=%d", res.Code, test.status)
		}
	}
}
