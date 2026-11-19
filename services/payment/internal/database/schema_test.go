package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPaymentAttemptConstraints(t *testing.T) {
	db := openSchemaTestDB(t)
	for _, test := range []struct {
		name   string
		update string
		state  string
	}{
		{"zero amount", "amount_idr = 0", "23514"},
		{"negative amount", "amount_idr = -1", "23514"},
		{"currency", "currency = 'USD'", "23514"},
		{"provider", "provider = 'other'", "23514"},
		{"bank", "bank = 'other'", "23514"},
		{"status", "status = 'unknown'", "23514"},
		{"paid without verification", "status = 'paid'", "23514"},
		{"empty order", "provider_order_id = ' '", "23514"},
		{"long order", "provider_order_id = repeat('a', 51)", "22001"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := schemaTestTransaction(t, db)
			id := insertTestAttempt(t, tx)
			// The SET clauses above are fixed test cases, never request input.
			_, err := tx.ExecContext(t.Context(), "UPDATE payment.payment_attempts SET "+test.update+" WHERE id = $1", id)
			requireSQLState(t, err, test.state)
		})
	}
}

func TestPaymentAttemptReferencesAndTimes(t *testing.T) {
	tx := schemaTestTransaction(t, openSchemaTestDB(t))
	id := insertTestAttempt(t, tx)
	var status, bank string
	var providerTime, verifiedTime sql.NullTime
	if err := tx.QueryRowContext(t.Context(), `SELECT status, bank, provider_paid_at, verified_at
		FROM payment.payment_attempts WHERE id = $1`, id).Scan(&status, &bank, &providerTime, &verifiedTime); err != nil {
		t.Fatal(err)
	}
	if status != "initiating" || bank != "bni" || providerTime.Valid || verifiedTime.Valid {
		t.Fatal("new attempts must allow unknown provider and verification times")
	}
	for _, status := range []string{"pending", "expired", "failed"} {
		if _, err := tx.ExecContext(t.Context(), "UPDATE payment.payment_attempts SET status = $1 WHERE id = $2", status, id); err != nil {
			t.Fatalf("supported status %s: %v", status, err)
		}
	}
	// A verified payment may need review without losing its paid fact.
	_, err := tx.ExecContext(t.Context(), `UPDATE payment.payment_attempts
		SET status = 'paid', verified_at = NOW(), review_reason = 'CUTOFF_UNCERTAIN'
		WHERE id = $1`, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRowContext(t.Context(), `SELECT status, provider_paid_at
		FROM payment.payment_attempts WHERE id = $1`, id).Scan(&status, &providerTime); err != nil {
		t.Fatal(err)
	}
	if status != "paid" || providerTime.Valid {
		t.Fatal("paid must remain readable when provider paid time is missing and review is required")
	}
	// These UUID references do not require users or bookings in another database.
	var externalForeignKeys int
	err = tx.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM pg_constraint c
		JOIN pg_class owned ON owned.oid = c.conrelid
		JOIN pg_namespace own_schema ON own_schema.oid = owned.relnamespace
		JOIN pg_class referenced ON referenced.oid = c.confrelid
		JOIN pg_namespace ref_schema ON ref_schema.oid = referenced.relnamespace
		WHERE c.contype = 'f' AND own_schema.nspname = 'payment' AND ref_schema.nspname <> 'payment'`).Scan(&externalForeignKeys)
	if err != nil || externalForeignKeys != 0 {
		t.Fatalf("external foreign keys = %d, error = %v", externalForeignKeys, err)
	}
}

func TestPaymentUniqueness(t *testing.T) {
	db := openSchemaTestDB(t)
	for _, test := range []struct {
		name   string
		column string
	}{
		{"one attempt per booking", "booking_id"},
		{"unique provider order", "provider_order_id"},
		{"unique provider transaction", "provider_transaction_id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := schemaTestTransaction(t, db)
			first := insertTestAttempt(t, tx)
			second := insertTestAttempt(t, tx)
			if test.column == "provider_transaction_id" {
				if _, err := tx.ExecContext(t.Context(), "UPDATE payment.payment_attempts SET provider_transaction_id = id::text WHERE id = $1", first); err != nil {
					t.Fatal(err)
				}
			}
			_, err := tx.ExecContext(t.Context(), `UPDATE payment.payment_attempts SET `+test.column+` =
				(SELECT `+test.column+` FROM payment.payment_attempts WHERE id = $1) WHERE id = $2`, first, second)
			requireSQLState(t, err, "23505")
		})
	}
}

func TestPaymentIdempotency(t *testing.T) {
	tx := schemaTestTransaction(t, openSchemaTestDB(t))
	id := insertTestAttempt(t, tx)
	for range 2 {
		_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.idempotency_records
			(customer_id, idempotency_key, request_hash, payment_id)
			SELECT customer_id, gen_random_uuid(), repeat('a', 64), id FROM payment.payment_attempts WHERE id = $1`, id)
		if err != nil {
			t.Fatalf("different keys must be able to reference the same attempt: %v", err)
		}
	}
	_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.idempotency_records
		(customer_id, idempotency_key, request_hash, payment_id)
		SELECT customer_id, idempotency_key, request_hash, payment_id FROM payment.idempotency_records WHERE payment_id = $1 LIMIT 1`, id)
	requireSQLState(t, err, "23505")
}

func TestPaymentAuditAndOutbox(t *testing.T) {
	tx := schemaTestTransaction(t, openSchemaTestDB(t))
	id := insertTestAttempt(t, tx)
	for _, source := range []string{"charge", "webhook", "status_lookup"} {
		_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.provider_observations
			(payment_id, source, verification_result, error_code)
			VALUES ($1, $2, 'unavailable', 'PROVIDER_TIMEOUT')`, id, source)
		if err != nil {
			t.Fatalf("unavailable observations may have no payload or paid timestamp: %v", err)
		}
	}
	if _, err := tx.ExecContext(t.Context(), `INSERT INTO payment.webhook_receipts (payload_hash, status)
		VALUES (repeat('b', 64), 'unmatched')`); err != nil {
		t.Fatalf("unmatched receipt may have no payment reference: %v", err)
	}
	if _, err := tx.ExecContext(t.Context(), `INSERT INTO payment.outbox_events
		(id, aggregate_id, event_type, envelope, occurred_at)
		VALUES (gen_random_uuid(), $1, 'payment.paid.v1', '{}', NOW())`, id); err != nil {
		t.Fatal(err)
	}
	_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.outbox_events
		(id, aggregate_id, event_type, envelope, occurred_at)
		VALUES (gen_random_uuid(), $1, 'payment.paid.v1', '{}', NOW())`, id)
	requireSQLState(t, err, "23505")
}

func TestPaymentChildConstraints(t *testing.T) {
	db := openSchemaTestDB(t)
	for _, test := range []struct {
		name  string
		query string
		state string
	}{
		{"idempotency hash", `INSERT INTO payment.idempotency_records (customer_id, idempotency_key, request_hash, payment_id) VALUES (gen_random_uuid(), gen_random_uuid(), 'short', $1)`, "23514"},
		{"idempotency reference", `INSERT INTO payment.idempotency_records (customer_id, idempotency_key, request_hash, payment_id) VALUES (gen_random_uuid(), gen_random_uuid(), repeat('a', 64), gen_random_uuid())`, "23503"},
		{"observation reference", `INSERT INTO payment.provider_observations (payment_id, source, verification_result) VALUES (gen_random_uuid(), 'charge', 'verified')`, "23503"},
		{"observation source", `INSERT INTO payment.provider_observations (payment_id, source, verification_result) VALUES ($1, 'invalid', 'verified')`, "23514"},
		{"observation verification", `INSERT INTO payment.provider_observations (payment_id, source, verification_result) VALUES ($1, 'charge', 'invalid')`, "23514"},
		{"receipt reference", `INSERT INTO payment.webhook_receipts (payment_id, payload_hash) VALUES (gen_random_uuid(), repeat('a', 64))`, "23503"},
		{"receipt status", `INSERT INTO payment.webhook_receipts (payload_hash, status) VALUES (repeat('a', 64), 'invalid')`, "23514"},
		{"outbox reference", `INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (gen_random_uuid(), gen_random_uuid(), 'payment.paid.v1', '{}', NOW())`, "23503"},
		{"outbox envelope", `INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (gen_random_uuid(), $1, 'payment.paid.v1', '[]', NOW())`, "23514"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tx := schemaTestTransaction(t, db)
			id := insertTestAttempt(t, tx)
			var err error
			// Only cases containing a parameter need the valid local payment ID.
			if strings.Contains(test.query, "$1") {
				_, err = tx.ExecContext(t.Context(), test.query, id)
			} else {
				_, err = tx.ExecContext(t.Context(), test.query)
			}
			requireSQLState(t, err, test.state)
		})
	}
}

func TestDuplicateWebhookReceipt(t *testing.T) {
	tx := schemaTestTransaction(t, openSchemaTestDB(t))
	for i := range 2 {
		_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.webhook_receipts (payload_hash)
			VALUES (repeat('c', 64))`)
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			requireSQLState(t, err, "23505")
		}
	}
}

func TestPaymentTransactionRollback(t *testing.T) {
	db := openSchemaTestDB(t)
	tx := schemaTestTransaction(t, db)
	id := insertTestAttempt(t, tx)
	if _, err := tx.ExecContext(t.Context(), `INSERT INTO payment.provider_observations
		(payment_id, source, verification_result) VALUES ($1, 'charge', 'verified')`, id); err != nil {
		t.Fatal(err)
	}
	_, err := tx.ExecContext(t.Context(), `INSERT INTO payment.outbox_events
		(id, aggregate_id, event_type, envelope, occurred_at)
		VALUES (gen_random_uuid(), $1, 'payment.paid.v1', '[]', NOW())`, id)
	requireSQLState(t, err, "23514")
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	err = db.QueryRowContext(t.Context(), `SELECT
		(SELECT COUNT(*) FROM payment.payment_attempts WHERE id = $1) +
		(SELECT COUNT(*) FROM payment.provider_observations WHERE payment_id = $1) +
		(SELECT COUNT(*) FROM payment.outbox_events WHERE aggregate_id = $1)`, id).Scan(&count)
	if err != nil || count != 0 {
		t.Fatalf("rollback left %d rows, error = %v", count, err)
	}
}

func TestConcurrentAttemptInsert(t *testing.T) {
	db := openSchemaTestDB(t)
	var bookingID string
	if err := db.QueryRowContext(t.Context(), "SELECT gen_random_uuid()").Scan(&bookingID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, "DELETE FROM payment.payment_attempts WHERE booking_id = $1", bookingID); err != nil {
			t.Error("could not clean up test attempt")
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			_, err := db.ExecContext(ctx, `INSERT INTO payment.payment_attempts
				(booking_id, customer_id, provider_order_id, amount_idr, hold_expires_at)
				VALUES ($1, gen_random_uuid(), gen_random_uuid()::text, 150000, NOW() + INTERVAL '15 minutes')`, bookingID)
			results <- err
		}()
	}
	close(start)
	winners, conflicts := 0, 0
	for range 2 {
		err := <-results
		var pgError *pgconn.PgError
		switch {
		case err == nil:
			winners++
		case errors.As(err, &pgError) && pgError.Code == "23505":
			conflicts++
		default:
			t.Fatalf("unexpected insert error: %v", err)
		}
	}
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM payment.payment_attempts WHERE booking_id = $1", bookingID).Scan(&count)
	if err != nil || winners != 1 || conflicts != 1 || count != 1 {
		t.Fatalf("winners=%d conflicts=%d stored=%d error=%v", winners, conflicts, count, err)
	}
}

func openSchemaTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYMENT_TEST_DATABASE_URL to a dedicated, migrated PostgreSQL test database")
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

func schemaTestTransaction(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	// Keep the transaction alive until cleanup explicitly rolls it back.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Error("could not roll back test transaction")
		}
	})
	return tx
}

func insertTestAttempt(t *testing.T, tx *sql.Tx) string {
	t.Helper()
	var id string
	err := tx.QueryRowContext(t.Context(), `INSERT INTO payment.payment_attempts
		(booking_id, customer_id, provider_order_id, amount_idr, hold_expires_at)
		VALUES (gen_random_uuid(), gen_random_uuid(), gen_random_uuid()::text, 150000, NOW() + INTERVAL '15 minutes')
		RETURNING id`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func requireSQLState(t *testing.T, err error, state string) {
	t.Helper()
	var pgError *pgconn.PgError
	if !errors.As(err, &pgError) || pgError.Code != state {
		t.Fatalf("error = %v, want SQLSTATE %s", err, state)
	}
}
