package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const demoPaymentBody = `{"amount_idr":800000,"currency":"IDR"}`

var demoPaymentTime = time.Date(2026, 11, 9, 4, 0, 0, 123456789, time.UTC)

func TestDemoConfirmationHTTP(t *testing.T) {
	enabled := demoConfirmationTestRouter(t, nil, paymentTestToken, true, demoPaymentTime)
	disabled := demoConfirmationTestRouter(t, nil, paymentTestToken, false, demoPaymentTime)
	noToken := demoConfirmationTestRouter(t, nil, "", true, demoPaymentTime)
	id := uuid.NewString()
	for _, test := range []struct {
		name, id, token, body, code string
		router                      http.Handler
		status                      int
	}{
		{"missing token", id, "", demoPaymentBody, "UNAUTHENTICATED", enabled, 401},
		{"customer token", id, "customer-fixture", demoPaymentBody, "UNAUTHENTICATED", enabled, 401},
		{"disabled", id, paymentTestToken, demoPaymentBody, "NOT_FOUND", disabled, 404},
		{"no configured token", id, paymentTestToken, demoPaymentBody, "NOT_FOUND", noToken, 404},
		{"invalid ID", "invalid", paymentTestToken, demoPaymentBody, "INVALID_REQUEST", enabled, 400},
		{"empty body", id, paymentTestToken, "", "INVALID_REQUEST", enabled, 400},
		{"null body", id, paymentTestToken, "null", "INVALID_REQUEST", enabled, 400},
		{"missing amount", id, paymentTestToken, `{"currency":"IDR"}`, "INVALID_REQUEST", enabled, 400},
		{"negative amount", id, paymentTestToken, `{"amount_idr":-1,"currency":"IDR"}`, "INVALID_REQUEST", enabled, 400},
		{"wrong currency", id, paymentTestToken, `{"amount_idr":800000,"currency":"USD"}`, "INVALID_REQUEST", enabled, 400},
		{"unknown field", id, paymentTestToken, `{"amount_idr":800000,"currency":"IDR","paid":true}`, "INVALID_REQUEST", enabled, 400},
		{"multiple objects", id, paymentTestToken, demoPaymentBody + demoPaymentBody, "INVALID_REQUEST", enabled, 400},
		{"body limit", id, paymentTestToken, strings.Repeat(" ", 16*1024+1), "PAYLOAD_TOO_LARGE", enabled, 413},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := demoConfirmationRequestForTest(test.router, test.id, test.token, test.body)
			assertDemoConfirmationError(t, res, test.status, test.code)
			if res.Header().Get("Cache-Control") != "no-store" || res.Header().Get("X-Request-ID") != "demo-confirmation-test" {
				t.Fatal("missing no-store or request ID")
			}
		})
	}
}

func TestDemoConfirmationPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	ctx := t.Context()
	bookingID := demoConfirmationBooking(t, pool, customerID)
	router := demoConfirmationTestRouter(t, pool, paymentTestToken, true, demoPaymentTime)
	res := demoConfirmationRequestForTest(router, bookingID, paymentTestToken, demoPaymentBody)
	var response struct{ Data checkout.DemoConfirmation }
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || res.Code != 200 {
		t.Fatalf("confirmation = %d %s (%v)", res.Code, res.Body.String(), err)
	}
	if response.Data.BookingID != bookingID || response.Data.Status != "confirmed" || !response.Data.Simulated ||
		!response.Data.ConfirmedAt.Equal(demoPaymentTime.Truncate(time.Microsecond)) {
		t.Fatalf("unexpected confirmation: %+v", response.Data)
	}
	var status, actor, reason, requestID, eventType string
	var blocking, simulated bool
	var amount int64
	var historyCount, eventCount int
	var updatedAt time.Time
	err := pool.QueryRowContext(ctx, `
		SELECT b.status, b.updated_at, a.blocks_inventory, h.actor_type, h.reason_code, h.request_id,
		       o.event_type, (o.envelope->'payload'->>'simulated')::boolean,
		       (o.envelope->'payload'->>'total_idr')::bigint,
		       (SELECT count(*) FROM booking.booking_status_history WHERE booking_id = b.id),
		       (SELECT count(*) FROM booking.outbox_events WHERE aggregate_id = b.id)
		FROM booking.bookings b JOIN booking.allocations a ON a.booking_id = b.id
		JOIN booking.booking_status_history h ON h.booking_id = b.id
		JOIN booking.outbox_events o ON o.aggregate_id = b.id WHERE b.id = $1
	`, bookingID).Scan(&status, &updatedAt, &blocking, &actor, &reason, &requestID, &eventType, &simulated, &amount, &historyCount, &eventCount)
	if err != nil || status != "confirmed" || !blocking || actor != "system" || reason != "DEMO_PAYMENT_CONFIRMED" ||
		requestID != "demo-confirmation-test" || eventType != "booking.demo_confirmed.v1" || !simulated || amount != 800000 ||
		historyCount != 1 || eventCount != 1 || !updatedAt.Equal(response.Data.ConfirmedAt) {
		t.Fatalf("unexpected persisted confirmation: %s %t %s %s %s %d %d %d (%v)", status, blocking, actor, reason, eventType, amount, historyCount, eventCount, err)
	}
	before := demoBookingSnapshot(t, pool, bookingID)
	// A new router/service and an elapsed deadline model a retry after restart.
	restarted := demoConfirmationTestRouter(t, pool, paymentTestToken, true, demoPaymentTime.Add(time.Hour))
	replay := demoConfirmationRequestForTest(restarted, strings.ToUpper(bookingID), paymentTestToken, demoPaymentBody)
	if replay.Code != 200 || replay.Body.String() != res.Body.String() {
		t.Fatalf("retry changed the response: %d %s", replay.Code, replay.Body.String())
	}
	wrongAmount := demoConfirmationRequestForTest(restarted, bookingID, paymentTestToken, `{"amount_idr":1,"currency":"IDR"}`)
	assertDemoConfirmationError(t, wrongAmount, 422, "PAYMENT_AMOUNT_MISMATCH")
	if before != demoBookingSnapshot(t, pool, bookingID) {
		t.Fatal("retry changed persisted state")
	}
	// A confirmed booking still excludes another hold on the same unit and schedule.
	var unitID string
	if err := pool.QueryRowContext(ctx, "SELECT unit_id FROM booking.allocations WHERE booking_id = $1", bookingID).Scan(&unitID); err != nil {
		t.Fatal(err)
	}
	if _, err := bookTestUnit(ctx, pool, customerID, unitID, pickup, returnTime); !hasSQLState(err, "23P01") {
		t.Fatalf("confirmed inventory no longer excluded: %v", err)
	}
}

func TestDemoConfirmationRejectionsPostgres(t *testing.T) {
	pool, customerID := demoConfirmationFixtures(t)
	for _, test := range []struct {
		name, setup, body, code string
		at                      time.Time
		status                  int
	}{
		{"amount mismatch", "", `{"amount_idr":1,"currency":"IDR"}`, "PAYMENT_AMOUNT_MISMATCH", demoPaymentTime, 422},
		{"at deadline", "", demoPaymentBody, "HOLD_EXPIRED", demoPaymentTime.Truncate(time.Microsecond).Add(15 * time.Minute), 409},
		{"after deadline", "", demoPaymentBody, "HOLD_EXPIRED", demoPaymentTime.Add(time.Hour), 409},
		{"expired", "UPDATE booking.bookings SET status = 'expired' WHERE id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"review", "UPDATE booking.bookings SET status = 'payment_review' WHERE id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"other confirmation", "UPDATE booking.bookings SET status = 'confirmed' WHERE id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"released", "UPDATE booking.allocations SET blocks_inventory = false, released_at = NOW() WHERE booking_id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"missing allocation", "DELETE FROM booking.allocations WHERE booking_id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"wrong interval", "UPDATE booking.allocations SET occupied_period = tstzrange('2026-12-01T00:00:00Z', '2026-12-02T00:00:00Z', '[)') WHERE booking_id = $1", demoPaymentBody, "BOOKING_NOT_CONFIRMABLE", demoPaymentTime, 409},
		{"rollback after writes", "INSERT INTO booking.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (gen_random_uuid(), $1, 'booking.demo_confirmed.v1', '{}', NOW())", demoPaymentBody, "DEPENDENCY_UNAVAILABLE", demoPaymentTime, 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := demoConfirmationBooking(t, pool, customerID)
			if test.setup != "" {
				if _, err := pool.ExecContext(t.Context(), test.setup, id); err != nil {
					t.Fatal(err)
				}
			}
			before := demoBookingSnapshot(t, pool, id)
			router := demoConfirmationTestRouter(t, pool, paymentTestToken, true, test.at)
			res := demoConfirmationRequestForTest(router, id, paymentTestToken, test.body)
			assertDemoConfirmationError(t, res, test.status, test.code)
			if before != demoBookingSnapshot(t, pool, id) {
				t.Fatal("rejected confirmation changed persisted data")
			}
		})
	}
	router := demoConfirmationTestRouter(t, pool, paymentTestToken, true, demoPaymentTime)
	res := demoConfirmationRequestForTest(router, uuid.NewString(), paymentTestToken, demoPaymentBody)
	assertDemoConfirmationError(t, res, 404, "BOOKING_NOT_FOUND")
}

func demoConfirmationFixtures(t *testing.T) (*sql.DB, string) {
	t.Helper()
	pool := openAllocationTestDB(t)
	customerID := uuid.NewString()
	cleanupTestBookings(t, pool, customerID)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, statement := range []string{
			"DELETE FROM booking.outbox_events WHERE aggregate_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.booking_status_history WHERE booking_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
		} {
			if _, err := pool.ExecContext(ctx, statement, customerID); err != nil {
				t.Error(err)
			}
		}
	})
	return pool, customerID
}

func demoConfirmationBooking(t *testing.T, pool *sql.DB, customerID string) string {
	t.Helper()
	id, err := bookTestUnit(t.Context(), pool, customerID, uuid.NewString(), pickup, returnTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.ExecContext(t.Context(), "UPDATE booking.bookings SET hold_expires_at = $1 WHERE id = $2", demoPaymentTime.Add(15*time.Minute), id); err != nil {
		t.Fatal(err)
	}
	return id
}

func demoConfirmationTestRouter(t *testing.T, pool *sql.DB, token string, enabled bool, at time.Time) *gin.Engine {
	t.Helper()
	router, err := newRouter(func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	var service *checkout.Service
	if pool != nil {
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Discard})
		if err != nil {
			t.Fatal(err)
		}
		service, err = checkout.NewService(nil, checkout.NewRepository(db))
		if err != nil {
			t.Fatal(err)
		}
	}
	registerDemoConfirmation(router, service, token, enabled, func() time.Time { return at })
	return router
}

func demoConfirmationRequestForTest(router http.Handler, id, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/bookings/"+id+"/demo-confirmation", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "demo-confirmation-test")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func assertDemoConfirmationError(t *testing.T, res *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if res.Code != status || !strings.Contains(res.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("response = %d %s, want %d %s", res.Code, res.Body.String(), status, code)
	}
}

func demoBookingSnapshot(t *testing.T, pool *sql.DB, id string) string {
	t.Helper()
	var snapshot string
	err := pool.QueryRowContext(t.Context(), `SELECT json_build_object(
		'booking', (SELECT row_to_json(b) FROM booking.bookings b WHERE id = $1),
		'allocations', (SELECT json_agg(a ORDER BY a.id) FROM booking.allocations a WHERE booking_id = $1),
		'history', (SELECT json_agg(h ORDER BY h.id) FROM booking.booking_status_history h WHERE booking_id = $1),
		'outbox', (SELECT json_agg(o ORDER BY o.id) FROM booking.outbox_events o WHERE aggregate_id = $1)
	)::text`, id).Scan(&snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}
