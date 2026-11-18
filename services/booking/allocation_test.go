package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	pickup     = "2026-11-10T10:00:00+08:00"
	returnTime = "2026-11-11T12:00:00+08:00"
	nextReturn = "2026-11-12T14:00:00+08:00"
)

func TestAllocationSchema(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	customerID := newTestID(t, ctx, db)
	firstUnit := newTestID(t, ctx, db)
	otherUnit := newTestID(t, ctx, db)
	cleanupTestBookings(t, db, customerID)

	firstBooking, err := bookTestUnit(ctx, db, customerID, firstUnit, pickup, returnTime)
	if err != nil {
		t.Fatalf("first allocation: %v", err)
	}

	// An overdue hold continues to block until the allocation is released.
	_, err = bookTestUnit(ctx, db, customerID, firstUnit, "2026-11-11T11:59:00+08:00", nextReturn)
	if !hasSQLState(err, "23P01") {
		t.Fatalf("overlapping allocation: got %v, want SQLSTATE 23P01", err)
	}
	if _, err := bookTestUnit(ctx, db, customerID, firstUnit, returnTime, nextReturn); err != nil {
		t.Fatalf("adjacent allocation: %v", err)
	}
	if _, err := bookTestUnit(ctx, db, customerID, otherUnit, pickup, returnTime); err != nil {
		t.Fatalf("overlapping allocation on another unit: %v", err)
	}

	_, err = db.ExecContext(ctx, `UPDATE booking.bookings SET total_idr = 1 WHERE id = $1`, firstBooking)
	if !hasSQLState(err, "23514") {
		t.Fatalf("incorrect booking total: got %v, want SQLSTATE 23514", err)
	}
	_, err = db.ExecContext(ctx, `UPDATE booking.allocations
		SET blocks_inventory = FALSE, released_at = NOW() WHERE booking_id = $1`, firstBooking)
	if err != nil {
		t.Fatalf("release allocation: %v", err)
	}
	if _, err := bookTestUnit(ctx, db, customerID, firstUnit, pickup, returnTime); err != nil {
		t.Fatalf("reserve released interval: %v", err)
	}
}

func TestConcurrentAllocation(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	customerID := newTestID(t, ctx, db)
	unitID := newTestID(t, ctx, db)
	cleanupTestBookings(t, db, customerID)

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := bookTestUnit(ctx, db, customerID, unitID, pickup, returnTime)
			results <- err
		}()
	}
	close(start)

	winners, conflicts := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			winners++
		case hasSQLState(err, "23P01"):
			conflicts++
		default:
			t.Errorf("unexpected allocation error: %v", err)
		}
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("got %d winners and %d conflicts, want one of each", winners, conflicts)
	}

	var bookings, allocations int
	err := db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM booking.bookings WHERE customer_id = $1),
		       (SELECT COUNT(*) FROM booking.allocations WHERE unit_id = $2 AND blocks_inventory)
	`, customerID, unitID).Scan(&bookings, &allocations)
	if err != nil {
		t.Fatal(err)
	}
	if bookings != 1 || allocations != 1 {
		t.Fatalf("stored %d bookings and %d blocking allocations, want one of each", bookings, allocations)
	}
}

func openAllocationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("BOOKING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BOOKING_TEST_DATABASE_URL to a dedicated, migrated PostgreSQL test database")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("could not open PostgreSQL test database")
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error("could not close PostgreSQL test database")
		}
	})
	return db
}

func newTestID(t *testing.T, ctx context.Context, db *sql.DB) string {
	t.Helper()
	var id string
	if err := db.QueryRowContext(ctx, "SELECT gen_random_uuid()").Scan(&id); err != nil {
		t.Fatal("could not generate UUID in PostgreSQL test database")
	}
	return id
}

func cleanupTestBookings(t *testing.T, db *sql.DB, customerID string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := db.ExecContext(ctx, `DELETE FROM booking.allocations WHERE booking_id IN
			(SELECT id FROM booking.bookings WHERE customer_id = $1)`, customerID)
		if err != nil {
			t.Errorf("cleanup allocations: %v", err)
			return
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM booking.bookings WHERE customer_id = $1`, customerID); err != nil {
			t.Errorf("cleanup bookings: %v", err)
		}
	})
}

func bookTestUnit(ctx context.Context, db *sql.DB, customerID, unitID, pickupAt, returnAt string) (string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var bookingID string
	err = tx.QueryRowContext(ctx, `
		WITH dates AS (
		    SELECT ($2::text::timestamptz AT TIME ZONE 'Asia/Makassar')::date AS pickup_date,
		           ($3::text::timestamptz AT TIME ZONE 'Asia/Makassar')::date AS return_date
		)
		INSERT INTO booking.bookings (
		    customer_id, variant_id, status, pickup_at, return_at, pickup_local_date, return_local_date,
		    vehicle_label_snapshot, daily_rate_idr, rental_days, total_idr,
		    terms_version, terms_accepted_at, hold_expires_at, created_at
		)
		SELECT $1, '00000000-0000-4000-8000-000000000001', 'pending_payment', $2::text::timestamptz, $3::text::timestamptz,
		       pickup_date, return_date, 'Test car', 400000, return_date - pickup_date + 1,
		       400000 * (return_date - pickup_date + 1),
		       'test-v1', NOW() - INTERVAL '20 minutes', NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '20 minutes'
		FROM dates
		RETURNING id
	`, customerID, pickupAt, returnAt).Scan(&bookingID)
	if err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO booking.allocations (booking_id, unit_id, occupied_period)
		VALUES ($1, $2, tstzrange($3::text::timestamptz, $4::text::timestamptz, '[)'))
	`, bookingID, unitID, pickupAt, returnAt)
	if err != nil {
		return "", err
	}
	return bookingID, tx.Commit()
}

func hasSQLState(err error, state string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == state
}
