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

const paymentTestToken = "payment-fixture-token"

func TestPaymentContextAuthorization(t *testing.T) {
	enabled := paymentContextTestRouter(t, nil, paymentTestToken)
	disabled := paymentContextTestRouter(t, nil, "")
	path := "/internal/v1/bookings/" + uuid.NewString() + "/payment-context"
	for _, test := range []struct {
		name, path, token, code string
		router                  http.Handler
		status                  int
	}{
		{"missing token", path, "", "UNAUTHENTICATED", enabled, 401},
		{"wrong token", path, "customer-fixture", "UNAUTHENTICATED", enabled, 401},
		{"invalid ID", "/internal/v1/bookings/invalid/payment-context", paymentTestToken, "INVALID_REQUEST", enabled, 400},
		{"disabled route", path, paymentTestToken, "NOT_FOUND", disabled, 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			res := paymentContextRequest(test.router, test.path, test.token)
			if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) ||
				res.Header().Get("X-Request-ID") != "payment-context-test" || res.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected response: %d %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestPaymentContextPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	customerID := uuid.NewString()
	cleanupTestBookings(t, db, customerID)
	bookingID, err := bookTestUnit(ctx, db, customerID, uuid.NewString(), pickup, returnTime)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Date(2026, 11, 9, 4, 15, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, "UPDATE booking.bookings SET hold_expires_at = $1 WHERE id = $2", deadline, bookingID); err != nil {
		t.Fatal(err)
	}
	router := paymentContextTestRouter(t, db, paymentTestToken)
	res := paymentContextRequest(router, "/internal/v1/bookings/"+bookingID+"/payment-context", paymentTestToken)
	var response struct{ Data checkout.PaymentContext }
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || res.Code != 200 {
		t.Fatalf("unexpected response: %d %s (%v)", res.Code, res.Body.String(), err)
	}
	if response.Data.BookingID != bookingID || response.Data.CustomerID != customerID ||
		response.Data.TotalIDR != 800000 || response.Data.Currency != "IDR" ||
		response.Data.Status != "pending_payment" || !response.Data.HoldExpiresAt.Equal(deadline) {
		t.Fatalf("unexpected payment context: %+v", response.Data)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil || len(envelope["data"]) != 6 {
		t.Fatal("payment context must contain the six contracted fields")
	}
	missing := paymentContextRequest(router, "/internal/v1/bookings/"+uuid.NewString()+"/payment-context", paymentTestToken)
	if missing.Code != 404 || !strings.Contains(missing.Body.String(), `"code":"BOOKING_NOT_FOUND"`) {
		t.Fatalf("unexpected missing booking response: %d %s", missing.Code, missing.Body.String())
	}
	var status string
	var blocking bool
	if err := db.QueryRowContext(ctx, `
		SELECT b.status, a.blocks_inventory FROM booking.bookings b
		JOIN booking.allocations a ON a.booking_id = b.id WHERE b.id = $1
	`, bookingID).Scan(&status, &blocking); err != nil || status != "pending_payment" || !blocking {
		t.Fatalf("reading an overdue booking changed its state: %s %t (%v)", status, blocking, err)
	}
}

func TestDueHoldsPostgres(t *testing.T) {
	pool := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	customerID := uuid.NewString()
	cleanupTestBookings(t, pool, customerID)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{
		DisableAutomaticPing: true, Logger: logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	repository := checkout.NewRepository(gormDB)
	checkedAt := time.Date(2000, 1, 1, 0, 15, 0, 0, time.UTC)
	expected := []string{}
	// Fifty-five due holds exercise the batch limit. Five share the cutoff exactly.
	for index := range 59 {
		bookingID, err := bookTestUnit(ctx, pool, customerID, uuid.NewString(), pickup, returnTime)
		if err != nil {
			t.Fatal(err)
		}
		if index < 50 {
			expected = append(expected, bookingID)
		}
		deadline := checkedAt.Add(-time.Duration(50-index) * time.Second)
		status := "pending_payment"
		switch index {
		case 55:
			deadline = checkedAt.Add(time.Second)
		case 56:
			status = "confirmed"
		case 57:
			status = "expired"
		case 58:
			status = "payment_review"
		default:
			if index >= 50 {
				deadline = checkedAt
			}
		}
		if _, err := pool.ExecContext(ctx, "UPDATE booking.bookings SET status = $1, hold_expires_at = $2 WHERE id = $3", status, deadline, bookingID); err != nil {
			t.Fatal(err)
		}
	}
	var before string
	const snapshotSQL = `
		SELECT json_build_object(
		    'bookings', (SELECT json_agg(b ORDER BY b.id) FROM booking.bookings b),
		    'allocations', (SELECT json_agg(a ORDER BY a.id) FROM booking.allocations a),
		    'history', (SELECT json_agg(h ORDER BY h.id) FROM booking.booking_status_history h),
		    'outbox', (SELECT json_agg(o ORDER BY o.id) FROM booking.outbox_events o)
		)::text
	`
	if err := pool.QueryRowContext(ctx, snapshotSQL).Scan(&before); err != nil {
		t.Fatal(err)
	}
	candidates, err := repository.DueHolds(ctx, checkedAt)
	if err != nil || len(candidates) != 50 {
		t.Fatalf("due holds = %d, want 50 (%v)", len(candidates), err)
	}
	for _, candidate := range candidates {
		if candidate.Status != "pending_payment" || candidate.HoldExpiresAt.After(checkedAt) {
			t.Fatalf("ineligible due candidate: %+v", candidate)
		}
	}
	for index, candidate := range candidates {
		if candidate.BookingID != expected[index] {
			t.Fatal("candidate order differs from deadline then ID")
		}
	}
	// Narrow to the five records at the cutoff to verify <= and tie ordering.
	if _, err := pool.ExecContext(ctx, `UPDATE booking.bookings SET status = 'confirmed'
		WHERE customer_id = $1 AND hold_expires_at < $2`, customerID, checkedAt); err != nil {
		t.Fatal(err)
	}
	atCutoff, err := repository.DueHolds(ctx, checkedAt)
	if err != nil || len(atCutoff) != 5 {
		t.Fatalf("cutoff candidates = %d, want 5 (%v)", len(atCutoff), err)
	}
	for index := 1; index < len(atCutoff); index++ {
		if atCutoff[index-1].BookingID >= atCutoff[index].BookingID {
			t.Fatal("equal-deadline candidates are not ordered by ID")
		}
	}
	empty, err := repository.DueHolds(ctx, checkedAt.Add(-time.Hour))
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("expected empty candidates before deadlines, got %v (%v)", empty, err)
	}
	// Restore our explicit fixture update before comparing the read operations.
	if _, err := pool.ExecContext(ctx, `UPDATE booking.bookings SET status = 'pending_payment'
		WHERE customer_id = $1 AND hold_expires_at < $2`, customerID, checkedAt); err != nil {
		t.Fatal(err)
	}
	var after string
	if err := pool.QueryRowContext(ctx, snapshotSQL).Scan(&after); err != nil || before != after {
		t.Fatalf("candidate detection changed persisted data (%v)", err)
	}
}

func paymentContextTestRouter(t *testing.T, pool *sql.DB, token string) *gin.Engine {
	t.Helper()
	router, err := newRouter(func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	var service *checkout.Service
	if pool != nil {
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{
			DisableAutomaticPing: true, Logger: logger.Discard,
		})
		if err != nil {
			t.Fatal(err)
		}
		service, err = checkout.NewService(nil, checkout.NewRepository(db))
		if err != nil {
			t.Fatal(err)
		}
	}
	registerPaymentContext(router, service, token)
	return router
}

func paymentContextRequest(router http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Request-ID", "payment-context-test")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}
