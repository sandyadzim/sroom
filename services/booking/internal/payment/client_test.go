package payment

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCanonicalConfirmationHTTP(t *testing.T) {
	const bookingID = "00000000-0000-4000-8000-000000000001"
	verified := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	for _, status := range []int{200, 401, 404, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/internal/v1/bookings/"+bookingID+"/payment-confirmation" || r.Header.Get("Authorization") != "Bearer synthetic-pair" || r.Header.Get("X-Request-ID") != "confirmation-test" {
					t.Error("incorrect private request")
				}
				w.WriteHeader(status)
				if status == 200 {
					_ = json.NewEncoder(w).Encode(map[string]any{"data": Confirmation{PaymentID: "00000000-0000-4000-8000-000000000002", BookingID: bookingID, CustomerID: "00000000-0000-4000-8000-000000000003", ProviderOrderID: "synthetic-order", AmountIDR: 150000, Currency: "IDR", Status: "paid", VerifiedAt: &verified, HoldExpiresAt: verified.Add(time.Minute)}})
				}
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "synthetic-pair")
			if err != nil {
				t.Fatal(err)
			}
			data, err := client.Confirmation(t.Context(), bookingID, "confirmation-test")
			if status == 200 {
				if err != nil || data.PaymentID == "" || data.VerifiedAt == nil {
					t.Fatal("valid canonical response rejected")
				}
			} else if status == 404 {
				if !errors.Is(err, ErrNotFound) {
					t.Fatal("missing attempt not handled")
				}
			} else if !errors.Is(err, ErrUnavailable) {
				t.Fatal("dependency failure not handled")
			}
		})
	}
}
