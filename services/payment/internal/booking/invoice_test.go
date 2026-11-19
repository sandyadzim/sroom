package booking

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInvoiceContextClient(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	at := time.Date(2030, 1, 1, 2, 0, 0, 0, time.UTC)
	base := InvoiceContext{BookingID: id, CustomerID: id, VariantID: id, PaymentID: id, ConfirmationEventID: id, ProviderOrderID: "synthetic-order", VehicleLabel: "Synthetic car", Status: "confirmed", ConfirmedAt: at, PickupAt: at.Add(24 * time.Hour), ReturnAt: at.Add(26 * time.Hour), RentalDays: 1, DailyRateIDR: 150000, TotalIDR: 150000, Currency: "IDR"}
	for _, name := range []string{"valid", "not found", "not confirmed", "unavailable", "wrong id", "wrong total", "missing reference"} {
		t.Run(name, func(t *testing.T) {
			data, status, want := base, 200, error(nil)
			switch name {
			case "not found":
				status, want = 404, ErrNotFound
			case "not confirmed":
				status, want = 409, ErrNotConfirmed
			case "unavailable":
				status, want = 503, ErrUnavailable
			case "wrong id":
				data.BookingID = "bad"
				want = ErrUnavailable
			case "wrong total":
				data.TotalIDR++
				want = ErrUnavailable
			case "missing reference":
				data.ConfirmationEventID = ""
				want = ErrUnavailable
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/internal/v1/bookings/"+id+"/invoice-context" || r.Header.Get("Authorization") != "Bearer synthetic-pair" || r.Header.Get("X-Request-ID") != "invoice-client" {
					t.Error("invalid invoice context request")
				}
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "synthetic-pair")
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.InvoiceContext(t.Context(), id, "invoice-client")
			if !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
		})
	}
}
