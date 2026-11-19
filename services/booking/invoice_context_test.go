package main

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"booking/internal/checkout"
)

func TestInvoiceContextAuthorization(t *testing.T) {
	app, err := newRouter(func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	registerInvoiceContext(app, nil, paymentTestToken)
	for _, test := range []struct {
		path, token string
		status      int
	}{
		{"00000000-0000-4000-8000-000000000001", "", 401}, {"bad", paymentTestToken, 400}, {"00000000-0000-4000-8000-000000000001?forged=1", paymentTestToken, 400},
	} {
		path := "/internal/v1/bookings/" + test.path + "/invoice-context"
		if test.status == 400 && test.path != "bad" {
			path = "/internal/v1/bookings/00000000-0000-4000-8000-000000000001/invoice-context?forged=1"
		}
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+test.token)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != test.status {
			t.Fatalf("status=%d want=%d", res.Code, test.status)
		}
	}
}

func TestInvoiceContextRejectsDemoPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	id := demoConfirmationBooking(t, pool, customerID)
	service := demoExpiryTestService(t, pool)
	app, err := newRouter(func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	registerInvoiceContext(app, service, paymentTestToken)
	for _, status := range []string{"pending_payment", "payment_review", "confirmed"} {
		if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET status = $1 WHERE id = $2", status, id); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/internal/v1/bookings/"+id+"/invoice-context", nil)
		req.Header.Set("Authorization", "Bearer "+paymentTestToken)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != 409 {
			t.Fatalf("unverified context status=%d", res.Code)
		}
		var response struct {
			Data checkout.InvoiceContext `json:"data"`
		}
		_ = json.Unmarshal(res.Body.Bytes(), &response)
		if response.Data.BookingID != "" {
			t.Fatal("unverified invoice snapshot returned")
		}
	}
}
