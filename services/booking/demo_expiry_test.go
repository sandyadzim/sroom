package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDemoExpirySchedule(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		ticks := make(chan time.Time, 4)
		done := make(chan struct{})
		start := time.Now()
		go func() {
			defer close(done)
			runExpiryChecks(ctx, func(context.Context) { ticks <- time.Now() })
		}()
		if first := <-ticks; !first.Equal(start) {
			t.Fatal("expiry must run immediately at startup")
		}
		time.Sleep(29 * time.Second)
		synctest.Wait()
		select {
		case <-ticks:
			t.Fatal("expiry ran before the 30-second interval")
		default:
		}
		time.Sleep(time.Second)
		if second := <-ticks; !second.Equal(start.Add(30 * time.Second)) {
			t.Fatal("expiry must run every 30 seconds")
		}
		cancel()
		<-done
		time.Sleep(time.Minute)
		select {
		case <-ticks:
			t.Fatal("expiry continued after shutdown")
		default:
		}
	})
}

func TestDemoExpiryWaitsForBatchShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			runExpiryChecks(ctx, func(ctx context.Context) {
				close(started)
				<-ctx.Done()
			})
		}()
		<-started
		cancel()
		<-done
	})
}

func TestDemoExpiryDisabled(t *testing.T) {
	for _, cfg := range []config{{}, {demoPayments: true}, {paymentToken: paymentTestToken}} {
		stop := startDemoExpiry(t.Context(), nil, cfg, slog.New(slog.NewJSONHandler(io.Discard, nil)))
		stop()
	}
}

func TestDemoExpiryStartupPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	service := demoExpiryTestService(t, pool)
	id := demoConfirmationBooking(t, pool, customerID)
	if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET hold_expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	stop := startDemoExpiry(t.Context(), service, config{demoPayments: true, paymentToken: paymentTestToken},
		slog.New(slog.NewJSONHandler(io.Discard, nil)))
	t.Cleanup(stop)
	deadline := time.Now().Add(3 * time.Second)
	for {
		var status string
		if err := pool.QueryRowContext(t.Context(), "SELECT status FROM booking.bookings WHERE id = $1", id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status == "expired" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("startup worker did not expire the overdue hold")
		}
		time.Sleep(5 * time.Millisecond)
	}
	stop()
}

func TestDemoExpiryPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	service := demoExpiryTestService(t, pool)
	bookingID := demoConfirmationBooking(t, pool, customerID)
	deadline := demoPaymentTime.Truncate(time.Microsecond).Add(15 * time.Minute)
	before := demoBookingSnapshot(t, pool, bookingID)
	result, err := service.ExpireDemoHolds(t.Context(), func() time.Time { return deadline.Add(-time.Nanosecond) }, "expiry-before")
	if err != nil || result.Expired != 0 || result.Failed != 0 || before != demoBookingSnapshot(t, pool, bookingID) {
		t.Fatalf("expiry before deadline changed state: %+v (%v)", result, err)
	}
	result, err = service.ExpireDemoHolds(t.Context(), func() time.Time { return deadline }, "expiry-test")
	if err != nil || result.Expired != 1 || result.Failed != 0 {
		t.Fatalf("expiry at deadline = %+v (%v)", result, err)
	}
	var status, actor, reason, requestID, unitID, eventType string
	var blocking bool
	var releasedAt, updatedAt time.Time
	var envelope struct {
		EventType     string `json:"event_type"`
		CorrelationID string `json:"correlation_id"`
		Payload       struct {
			BookingID  string    `json:"booking_id"`
			ExpiredAt  time.Time `json:"expired_at"`
			ReasonCode string    `json:"reason_code"`
			Simulated  bool      `json:"simulated"`
		} `json:"payload"`
	}
	var eventJSON []byte
	err = pool.QueryRowContext(t.Context(), `
		SELECT b.status, b.updated_at, a.unit_id, a.blocks_inventory, a.released_at,
		       h.actor_type, h.reason_code, h.request_id, o.event_type, o.envelope
		FROM booking.bookings b JOIN booking.allocations a ON a.booking_id = b.id
		JOIN booking.booking_status_history h ON h.booking_id = b.id
		JOIN booking.outbox_events o ON o.aggregate_id = b.id WHERE b.id = $1
	`, bookingID).Scan(&status, &updatedAt, &unitID, &blocking, &releasedAt, &actor, &reason, &requestID, &eventType, &eventJSON)
	if err != nil || status != "expired" || blocking || !releasedAt.Equal(deadline) || !updatedAt.Equal(deadline) ||
		actor != "system" || reason != "DEMO_HOLD_EXPIRED" || requestID != "expiry-test" || eventType != "booking.demo_expired.v1" {
		t.Fatalf("unexpected persisted expiry: %s %t %s %s %s (%v)", status, blocking, actor, reason, eventType, err)
	}
	if err := json.Unmarshal(eventJSON, &envelope); err != nil || envelope.EventType != eventType || envelope.CorrelationID != requestID ||
		envelope.Payload.BookingID != bookingID || envelope.Payload.ReasonCode != reason || !envelope.Payload.Simulated || !envelope.Payload.ExpiredAt.Equal(deadline) {
		t.Fatalf("unexpected demo expiry event: %s (%v)", eventJSON, err)
	}
	expiredSnapshot := demoBookingSnapshot(t, pool, bookingID)
	restarted := demoExpiryTestService(t, pool)
	result, err = restarted.ExpireDemoHolds(t.Context(), func() time.Time { return deadline.Add(time.Hour) }, "expiry-retry")
	if err != nil || result.Expired != 0 || expiredSnapshot != demoBookingSnapshot(t, pool, bookingID) {
		t.Fatalf("restart/retry changed an expired booking: %+v (%v)", result, err)
	}
	// Inventory can be reserved again; a late demo confirmation cannot take it back.
	if _, err := bookTestUnit(t.Context(), pool, customerID, unitID, pickup, returnTime); err != nil {
		t.Fatalf("could not reserve the released interval: %v", err)
	}
	router := demoConfirmationTestRouter(t, pool, paymentTestToken, true, deadline.Add(time.Minute))
	late := demoConfirmationRequestForTest(router, bookingID, paymentTestToken, demoPaymentBody)
	assertDemoConfirmationError(t, late, 409, "BOOKING_NOT_CONFIRMABLE")
	if expiredSnapshot != demoBookingSnapshot(t, pool, bookingID) {
		t.Fatal("late confirmation changed released inventory")
	}
	parsedCustomer, err := uuid.Parse(customerID)
	if err != nil {
		t.Fatal(err)
	}
	booking, err := restarted.CustomerBooking(t.Context(), parsedCustomer, bookingID)
	if err != nil || booking.Status != "expired" {
		t.Fatalf("customer read did not show expired status: %+v (%v)", booking, err)
	}
}

func TestDemoExpiryInvalidAllocationsPostgres(t *testing.T) {
	for _, test := range []struct{ name, setup string }{
		{"missing", "DELETE FROM booking.allocations WHERE booking_id = $1"},
		{"released", "UPDATE booking.allocations SET blocks_inventory = false, released_at = NOW() WHERE booking_id = $1"},
		{"wrong interval", "UPDATE booking.allocations SET occupied_period = tstzrange('2026-12-01T00:00:00Z', '2026-12-02T00:00:00Z', '[)') WHERE booking_id = $1"},
		{"outbox rollback", "INSERT INTO booking.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (gen_random_uuid(), $1, 'booking.demo_expired.v1', '{}', NOW())"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool, customerID := demoConfirmationFixtures(t)
			badID := demoConfirmationBooking(t, pool, customerID)
			goodID := demoConfirmationBooking(t, pool, customerID)
			if _, err := pool.ExecContext(t.Context(), test.setup, badID); err != nil {
				t.Fatal(err)
			}
			// Ensure the failing candidate is visited first and does not prevent the good one.
			if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET hold_expires_at = hold_expires_at - INTERVAL '1 second' WHERE id = $1", badID); err != nil {
				t.Fatal(err)
			}
			before := demoBookingSnapshot(t, pool, badID)
			service := demoExpiryTestService(t, pool)
			result, err := service.ExpireDemoHolds(t.Context(), func() time.Time { return demoPaymentTime.Add(time.Hour) }, "expiry-failure")
			if err == nil || result.Expired != 1 || result.Failed != 1 || before != demoBookingSnapshot(t, pool, badID) {
				t.Fatalf("failure did not roll back and continue: %+v (%v)", result, err)
			}
			var status string
			if err := pool.QueryRowContext(t.Context(), "SELECT status FROM booking.bookings WHERE id = $1", goodID).Scan(&status); err != nil || status != "expired" {
				t.Fatalf("good candidate was not expired: %s (%v)", status, err)
			}
		})
	}
}

func TestDemoExpiryBatchAndStatusesPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	for index := range 56 {
		id := demoConfirmationBooking(t, pool, customerID)
		if index >= 53 {
			status := []string{"confirmed", "expired", "payment_review"}[index-53]
			if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET status = $1 WHERE id = $2", status, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	service := demoExpiryTestService(t, pool)
	result, err := service.ExpireDemoHolds(t.Context(), func() time.Time { return demoPaymentTime.Add(time.Hour) }, "expiry-first-batch")
	if err != nil || result.Expired != 50 || result.Failed != 0 {
		t.Fatalf("first batch = %+v (%v)", result, err)
	}
	result, err = service.ExpireDemoHolds(t.Context(), func() time.Time { return demoPaymentTime.Add(time.Hour) }, "expiry-second-batch")
	if err != nil || result.Expired != 3 || result.Failed != 0 {
		t.Fatalf("second batch = %+v (%v)", result, err)
	}
	var protected int
	if err := pool.QueryRowContext(t.Context(), `SELECT count(*) FROM booking.bookings b
		JOIN booking.allocations a ON a.booking_id = b.id
		WHERE b.customer_id = $1 AND b.status IN ('confirmed', 'payment_review') AND a.blocks_inventory`, customerID).Scan(&protected); err != nil || protected != 2 {
		t.Fatalf("confirmed/review inventory changed: %d (%v)", protected, err)
	}
}

func TestDemoExpiryConfirmationRacePostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	service := demoExpiryTestService(t, pool)
	id := demoConfirmationBooking(t, pool, customerID)
	deadline := demoPaymentTime.Truncate(time.Microsecond).Add(15 * time.Minute)
	start := make(chan struct{})
	confirmed := make(chan error, 1)
	expired := make(chan error, 1)
	go func() {
		<-start
		_, err := service.ConfirmDemoPayment(t.Context(), id, 800000, "IDR", func() time.Time { return deadline.Add(-time.Microsecond) }, "race-confirm")
		confirmed <- err
	}()
	go func() {
		<-start
		_, err := service.ExpireDemoHolds(t.Context(), func() time.Time { return deadline }, "race-expire")
		expired <- err
	}()
	close(start)
	confirmationError, expiryError := <-confirmed, <-expired
	if expiryError != nil || (confirmationError != nil && !errors.Is(confirmationError, checkout.ErrBookingNotConfirmable)) {
		t.Fatalf("unexpected concurrent result: %v / %v", confirmationError, expiryError)
	}
	var status string
	var blocking bool
	var events, history int
	err := pool.QueryRowContext(t.Context(), `SELECT b.status, a.blocks_inventory,
		(SELECT count(*) FROM booking.outbox_events WHERE aggregate_id = b.id),
		(SELECT count(*) FROM booking.booking_status_history WHERE booking_id = b.id)
		FROM booking.bookings b JOIN booking.allocations a ON a.booking_id = b.id WHERE b.id = $1`, id).
		Scan(&status, &blocking, &events, &history)
	if err != nil || events != 1 || history != 1 || (status == "confirmed") != blocking ||
		(status != "confirmed" && status != "expired") || (confirmationError == nil) != (status == "confirmed") {
		t.Fatalf("inconsistent concurrent result: %s %t events=%d history=%d (%v)", status, blocking, events, history, err)
	}
}

func demoExpiryTestService(t *testing.T, pool *sql.DB) *checkout.Service {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	service, err := checkout.NewService(nil, checkout.NewRepository(db))
	if err != nil {
		t.Fatal(err)
	}
	return service
}
