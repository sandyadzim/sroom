package payment

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExpiryAssessmentClient(t *testing.T) {
	const bookingID = "00000000-0000-4000-8000-000000000001"
	const paymentID = "00000000-0000-4000-8000-000000000002"
	deadline := time.Date(2030, 1, 1, 0, 15, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		status int
		data   ExpiryAssessment
		want   error
	}{
		{"closed", 200, ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: deadline, Result: "no_attempt_closed", ClosedAt: &deadline}, nil},
		{"existing", 200, ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: deadline, Result: "attempt_exists", PaymentID: stringPointer(paymentID)}, nil},
		{"not eligible", 409, ExpiryAssessment{}, ErrNotExpirable}, {"missing", 404, ExpiryAssessment{}, ErrUnavailable}, {"down", 503, ExpiryAssessment{}, ErrUnavailable},
		{"unproved closure", 200, ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: deadline, Result: "no_attempt_closed"}, ErrUnavailable},
		{"wrong booking", 200, ExpiryAssessment{BookingID: paymentID, HoldExpiresAt: deadline, Result: "no_attempt_closed", ClosedAt: &deadline}, ErrUnavailable},
		{"unknown result", 200, ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: deadline, Result: "expired", ClosedAt: &deadline}, ErrUnavailable},
		{"attempt with closure", 200, ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: deadline, Result: "no_attempt_closed", ClosedAt: &deadline, PaymentID: stringPointer(paymentID)}, ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/internal/v1/bookings/"+bookingID+"/expiry-assessment" || r.Header.Get("Authorization") != "Bearer synthetic-pair" || r.Header.Get("X-Request-ID") != "expiry-test" || r.ContentLength != 0 {
					t.Error("invalid private assessment request")
				}
				w.WriteHeader(test.status)
				_ = json.NewEncoder(w).Encode(map[string]any{"data": test.data})
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "synthetic-pair")
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.AssessExpiry(t.Context(), bookingID, "expiry-test")
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
