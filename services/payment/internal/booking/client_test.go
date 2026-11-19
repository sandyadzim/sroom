package booking

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaymentContextClient(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   error
	}{
		{200, `{"data":{"booking_id":"booking-fixture","customer_id":"customer-fixture","total_idr":150000,"currency":"IDR","status":"pending_payment","hold_expires_at":"2026-11-20T02:15:00Z"}}`, nil},
		{404, `{}`, ErrNotFound},
		{401, `{"private":"should not escape"}`, ErrUnavailable},
		{503, `{}`, ErrUnavailable},
		{200, `broken`, ErrUnavailable},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" || r.URL.Path != "/internal/v1/bookings/booking-fixture/payment-context" || r.Header.Get("Authorization") != "Bearer test-only-pair-token" || r.Header.Get("X-Request-ID") != "booking-test" {
				t.Error("invalid private Booking request")
			}
			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		}))
		client, err := NewClient(server.URL, "test-only-pair-token")
		if err != nil {
			t.Fatal(err)
		}
		result, err := client.PaymentContext(t.Context(), "booking-fixture", "booking-test")
		server.Close()
		if !errors.Is(err, test.want) {
			t.Fatalf("error=%v want=%v", err, test.want)
		}
		if err == nil && (result.TotalIDR != 150000 || result.Currency != "IDR") {
			t.Fatal("incorrect Booking price")
		}
	}
}

func TestInvalidBookingOrigin(t *testing.T) {
	for _, url := range []string{"", "file:///tmp/file", "http://user:pass@localhost", "http://localhost/path", "http://localhost?token=value"} {
		if _, err := NewClient(url, "test"); err == nil {
			t.Fatal("invalid Booking origin accepted")
		}
	}
}
