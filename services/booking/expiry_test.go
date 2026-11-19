package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"booking/internal/checkout"
	"booking/internal/payment"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPaymentExpiryDisabled(t *testing.T) {
	for _, cfg := range []config{{}, {paymentToken: paymentTestToken}, {rabbitURL: "amqp://unused.invalid"}, {rabbitURL: "amqp://unused.invalid", paymentToken: paymentTestToken, demoPayments: true}} {
		stop := startPaymentExpiry(t.Context(), nil, cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
		stop()
	}
}

func TestPaymentExpiryStartupPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	id := demoConfirmationBooking(t, pool, customerID)
	deadline := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET hold_expires_at = $1 WHERE id = $2", deadline, id); err != nil {
		t.Fatal(err)
	}
	called := make(chan struct{}, 1)
	paymentAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer "+paymentTestToken {
			t.Error("incorrect worker assessment request")
			w.WriteHeader(401)
			return
		}
		if _, err := uuid.Parse(r.Header.Get("X-Request-ID")); err != nil {
			t.Error("missing correlation ID")
		}
		defer func() {
			select {
			case called <- struct{}{}:
			default:
			}
		}()
		// Other packages can have earlier due fixtures in the same test database.
		// Observe startup without changing their allocations or requiring a second tick.
		if r.URL.Path != "/internal/v1/bookings/"+id+"/expiry-assessment" {
			w.WriteHeader(409)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": payment.ExpiryAssessment{BookingID: id, HoldExpiresAt: deadline, Result: "no_attempt_closed", ClosedAt: &deadline}})
	}))
	defer paymentAPI.Close()
	client, err := payment.NewClient(paymentAPI.URL, paymentTestToken)
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	svc := checkout.NewExpiryService(checkout.NewRepository(db), client)
	stop := startPaymentExpiry(t.Context(), svc, config{rabbitURL: "amqp://unused.invalid", paymentToken: paymentTestToken}, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	defer stop()
	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not run at startup without a live broker")
	}
}
