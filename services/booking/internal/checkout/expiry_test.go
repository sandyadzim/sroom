package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"booking/internal/payment"

	"github.com/google/uuid"
)

type expiryReader struct {
	assess func(context.Context, string, string) (payment.ExpiryAssessment, error)
}

func (r expiryReader) AssessExpiry(ctx context.Context, id, requestID string) (payment.ExpiryAssessment, error) {
	return r.assess(ctx, id, requestID)
}

func unusedProof(f confirmationFixture) payment.ExpiryAssessment {
	deadline := f.reader.data.HoldExpiresAt
	return payment.ExpiryAssessment{BookingID: f.event.Payload.BookingID, HoldExpiresAt: deadline, Result: "no_attempt_closed", ClosedAt: &deadline}
}

func allocationBlocks(t *testing.T, f confirmationFixture) bool {
	t.Helper()
	var blocking bool
	if err := f.db.Raw("SELECT blocks_inventory FROM booking.allocations WHERE booking_id = ?", f.event.Payload.BookingID).Scan(&blocking).Error; err != nil {
		t.Fatal(err)
	}
	return blocking
}

func TestUnusedHoldExpiryAndRetryPostgres(t *testing.T) {
	f := paidFixture(t)
	proof := unusedProof(f)
	repo := NewRepository(f.db)
	for _, at := range []time.Time{proof.HoldExpiresAt.Add(-time.Microsecond), proof.HoldExpiresAt, proof.HoldExpiresAt.Add(time.Minute)} {
		expired, err := repo.expireUnusedHold(t.Context(), proof, func() time.Time { return at }, "expiry-test")
		if err != nil {
			t.Fatal(err)
		}
		if at.Before(proof.HoldExpiresAt) {
			if expired || !allocationBlocks(t, f) || bookingStatus(t, f) != "pending_payment" {
				t.Fatal("hold released before deadline")
			}
			continue
		}
		// A fresh repository simulates retry after a process restart.
		repo = NewRepository(f.db)
		if bookingStatus(t, f) != "expired" || allocationBlocks(t, f) || countRows(t, f, "booking_status_history", "booking_id") != 1 || countRows(t, f, "outbox_events", "aggregate_id") != 1 {
			t.Fatal("expiry effects were partial or duplicated")
		}
	}
	var recorded struct{ Envelope string }
	if err := f.db.Raw("SELECT envelope::text FROM booking.outbox_events WHERE aggregate_id = ?", proof.BookingID).Scan(&recorded).Error; err != nil {
		t.Fatal(err)
	}
	var event struct {
		EventType string `json:"event_type"`
		Payload   struct {
			BookingID          string    `json:"booking_id"`
			ReasonCode         string    `json:"reason_code"`
			InitiationClosedAt time.Time `json:"initiation_closed_at"`
		} `json:"payload"`
	}
	if json.Unmarshal([]byte(recorded.Envelope), &event) != nil || event.EventType != "booking.expired.v1" || event.Payload.BookingID != proof.BookingID || event.Payload.ReasonCode != "NO_PAYMENT_ATTEMPT_CLOSED" || !event.Payload.InitiationClosedAt.Equal(*proof.ClosedAt) {
		t.Fatal("expiry event lost its closure evidence")
	}
	var unit string
	if err := f.db.Raw("SELECT unit_id FROM booking.allocations WHERE booking_id = ?", proof.BookingID).Scan(&unit).Error; err != nil {
		t.Fatal(err)
	}
	available, err := repo.HasAvailableUnit(t.Context(), []string{unit}, time.Date(2030, 1, 11, 2, 0, 0, 0, time.UTC), time.Date(2030, 1, 12, 4, 0, 0, 0, time.UTC))
	if err != nil || !available {
		t.Fatal("released unit did not become available")
	}
}

func TestUnusedHoldExpiryGuardsPostgres(t *testing.T) {
	for _, name := range []string{"confirmed", "review", "attempt", "deadline mismatch", "missing proof", "released", "wrong period", "rollback"} {
		t.Run(name, func(t *testing.T) {
			f := paidFixture(t)
			proof := unusedProof(f)
			want := ErrInvalidExpiryAssessment
			switch name {
			case "confirmed", "review":
				status := "confirmed"
				if name == "review" {
					status = "payment_review"
				}
				if err := f.db.Exec("UPDATE booking.bookings SET status = ? WHERE id = ?", status, proof.BookingID).Error; err != nil {
					t.Fatal(err)
				}
				want = nil
			case "attempt":
				proof.Result = "attempt_exists"
			case "deadline mismatch":
				proof.HoldExpiresAt = proof.HoldExpiresAt.Add(-time.Second)
			case "missing proof":
				proof.ClosedAt = nil
			case "released":
				if err := f.db.Exec("UPDATE booking.allocations SET blocks_inventory = false, released_at = NOW() WHERE booking_id = ?", proof.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "wrong period":
				if err := f.db.Exec("UPDATE booking.allocations SET occupied_period = tstzrange('2030-01-11T11:00:00+08:00','2030-01-12T12:00:00+08:00','[)') WHERE booking_id = ?", proof.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "rollback":
				if err := f.db.Exec("INSERT INTO booking.outbox_events (id,aggregate_id,event_type,envelope,occurred_at) VALUES (?, ?, 'booking.expired.v1','{}',?)", uuid.NewString(), proof.BookingID, f.now).Error; err != nil {
					t.Fatal(err)
				}
				want = ErrDatabaseUnavailable
			}
			status := bookingStatus(t, f)
			expired, err := NewRepository(f.db).expireUnusedHold(t.Context(), proof, func() time.Time { return f.reader.data.HoldExpiresAt.Add(time.Minute) }, "expiry-guard")
			if !errors.Is(err, want) || expired || bookingStatus(t, f) != status || countRows(t, f, "booking_status_history", "booking_id") != 0 {
				t.Fatalf("guard failed: expired=%v error=%v", expired, err)
			}
			if allocationBlocks(t, f) != (name != "released") {
				t.Fatal("guard changed allocation")
			}
		})
	}
}

func TestExpiryWorkerRetainsAttemptsAndRecoversPostgres(t *testing.T) {
	f := paidFixture(t)
	proof := unusedProof(f)
	// An early synthetic deadline isolates this scanner from other packages' fixtures.
	deadline := time.Date(2000, 1, 1, 0, 15, 0, 0, time.UTC)
	if err := f.db.Exec("UPDATE booking.bookings SET hold_expires_at = ? WHERE id = ?", deadline, proof.BookingID).Error; err != nil {
		t.Fatal(err)
	}
	proof.HoldExpiresAt, proof.ClosedAt = deadline, &deadline
	var dependencyError error
	var hasAttempt = true
	reader := expiryReader{assess: func(_ context.Context, id, requestID string) (payment.ExpiryAssessment, error) {
		if id != proof.BookingID || requestID != "expiry-worker" {
			t.Fatal("incorrect expiry request")
		}
		if dependencyError != nil {
			return payment.ExpiryAssessment{}, dependencyError
		}
		if hasAttempt {
			result := proof
			result.Result, result.ClosedAt = "attempt_exists", nil
			result.PaymentID = &f.reader.data.PaymentID
			return result, nil
		}
		return proof, nil
	}}
	svc := NewExpiryService(NewRepository(f.db), reader)
	now := func() time.Time { return deadline }
	result, err := svc.ExpireHolds(t.Context(), now, "expiry-worker")
	if err != nil || result.Held != 1 || result.Expired != 0 || !allocationBlocks(t, f) || bookingStatus(t, f) != "pending_payment" {
		t.Fatal("attempt was released or moved to terminal review")
	}
	dependencyError = payment.ErrUnavailable
	result, err = svc.ExpireHolds(t.Context(), now, "expiry-worker")
	if !errors.Is(err, payment.ErrUnavailable) || result.Failed != 1 || !allocationBlocks(t, f) {
		t.Fatal("dependency failure released a hold")
	}
	dependencyError, hasAttempt = nil, false
	svc = NewExpiryService(NewRepository(f.db), reader)
	result, err = svc.ExpireHolds(t.Context(), now, "expiry-worker")
	if err != nil || result.Expired != 1 || bookingStatus(t, f) != "expired" {
		t.Fatal("worker did not resume after restart")
	}
}

func TestExpiryScanRotatesPastHeldBookingsPostgres(t *testing.T) {
	f := paidFixture(t)
	deadline := time.Date(2000, 1, 1, 0, 15, 0, 0, time.UTC)
	ids := []string{f.event.Payload.BookingID}
	t.Cleanup(func() {
		for _, id := range ids[1:] {
			for _, table := range []string{"outbox_events", "booking_status_history", "allocations", "bookings"} {
				column := "booking_id"
				if table == "outbox_events" {
					column = "aggregate_id"
				}
				if table == "bookings" {
					column = "id"
				}
				if err := f.db.Exec("DELETE FROM booking."+table+" WHERE "+column+" = ?", id).Error; err != nil {
					t.Error(err)
				}
			}
		}
	})
	if err := f.db.Exec("UPDATE booking.bookings SET hold_expires_at = ? WHERE id = ?", deadline, ids[0]).Error; err != nil {
		t.Fatal(err)
	}
	for range 50 {
		id := uuid.NewString()
		if err := f.db.Exec(`INSERT INTO booking.bookings
   (id,customer_id,variant_id,status,pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at)
   SELECT ?,customer_id,variant_id,status,pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at FROM booking.bookings WHERE id = ?`, id, ids[0]).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	seen := map[string]int{}
	reader := expiryReader{assess: func(_ context.Context, id, _ string) (payment.ExpiryAssessment, error) {
		seen[id]++
		return payment.ExpiryAssessment{BookingID: id, HoldExpiresAt: deadline, Result: "attempt_exists", PaymentID: &f.reader.data.PaymentID}, nil
	}}
	svc := NewExpiryService(NewRepository(f.db), reader)
	for i, want := range []int{50, 1, 50} {
		result, err := svc.ExpireHolds(t.Context(), func() time.Time { return deadline }, "scan")
		if err != nil || result.Held != want {
			t.Fatalf("batch %d: %+v error=%v", i, result, err)
		}
	}
	if len(seen) != 51 {
		t.Fatal("held bookings starved later candidates")
	}
}

func TestExpiryAndConfirmationSerializePostgres(t *testing.T) {
	f := paidFixture(t)
	repo, proof := NewRepository(f.db), unusedProof(f)
	start, results := make(chan struct{}), make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := repo.expireUnusedHold(t.Context(), proof, func() time.Time { return proof.HoldExpiresAt }, "expiry-race")
		results <- err
	}()
	go func() { defer wg.Done(); <-start; results <- f.service.Process(t.Context(), eventBody(t, f.event)) }()
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	status, blocking := bookingStatus(t, f), allocationBlocks(t, f)
	if (status != "confirmed" || !blocking) && (status != "payment_review" || blocking) {
		t.Fatalf("unsafe result: status=%s blocking=%v", status, blocking)
	}
}

func TestReleasedUnitIsNotReclaimedPostgres(t *testing.T) {
	f := paidFixture(t)
	proof, repo := unusedProof(f), NewRepository(f.db)
	if expired, err := repo.expireUnusedHold(t.Context(), proof, func() time.Time { return proof.HoldExpiresAt }, "release"); err != nil || !expired {
		t.Fatalf("release=%v error=%v", expired, err)
	}
	newID := uuid.NewString()
	t.Cleanup(func() {
		for _, query := range []string{"DELETE FROM booking.allocations WHERE booking_id = ?", "DELETE FROM booking.bookings WHERE id = ?"} {
			if err := f.db.Exec(query, newID).Error; err != nil {
				t.Error(err)
			}
		}
	})
	if err := f.db.Exec(`INSERT INTO booking.bookings
   (id,customer_id,variant_id,status,pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at)
   SELECT ?,customer_id,variant_id,'pending_payment',pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at FROM booking.bookings WHERE id = ?`, newID, proof.BookingID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Exec("INSERT INTO booking.allocations (booking_id,unit_id,occupied_period) SELECT ?,unit_id,occupied_period FROM booking.allocations WHERE booking_id = ?", newID, proof.BookingID).Error; err != nil {
		t.Fatal("released slot could not be booked again")
	}
	// A later paid fact never takes this unit back from the new booking.
	if err := f.service.Process(t.Context(), eventBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	var status string
	var blocking bool
	if err := f.db.Raw("SELECT b.status,a.blocks_inventory FROM booking.bookings b JOIN booking.allocations a ON a.booking_id = b.id WHERE b.id = ?", newID).Row().Scan(&status, &blocking); err != nil {
		t.Fatal(err)
	}
	if bookingStatus(t, f) != "payment_review" || allocationBlocks(t, f) || status != "pending_payment" || !blocking {
		t.Fatal("late payment reclaimed inventory")
	}
}

func TestProviderExpiredHoldReleasesOncePostgres(t *testing.T) {
	f := paidFixture(t)
	deadline := f.reader.data.HoldExpiresAt
	proof := payment.ExpiryAssessment{BookingID: f.event.Payload.BookingID, HoldExpiresAt: deadline,
		Result: "provider_expired", PaymentID: &f.reader.data.PaymentID, ReleaseAuthorizedAt: &deadline}
	repo := NewRepository(f.db)
	for _, want := range []bool{true, false} {
		expired, err := repo.expireUnusedHold(t.Context(), proof, func() time.Time { return deadline }, "provider-expiry")
		if err != nil || expired != want || bookingStatus(t, f) != "expired" || allocationBlocks(t, f) {
			t.Fatalf("expiry=%v error=%v", expired, err)
		}
	}
	var reason, paymentID string
	if err := f.db.Raw("SELECT envelope->'payload'->>'reason_code',envelope->'payload'->>'payment_id' FROM booking.outbox_events WHERE aggregate_id = ?", proof.BookingID).Row().Scan(&reason, &paymentID); err != nil {
		t.Fatal(err)
	}
	if reason != "PROVIDER_EXPIRED" || paymentID != *proof.PaymentID || countRows(t, f, "booking_status_history", "booking_id") != 1 {
		t.Fatal("release evidence or idempotence was lost")
	}
	// The existing confirmation path must send later payment to review without reclaiming inventory.
	f.reader.data.ReviewRequired = true
	if err := f.service.Process(t.Context(), eventBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	if bookingStatus(t, f) != "payment_review" || allocationBlocks(t, f) {
		t.Fatal("late payment reclaimed the released unit")
	}
}
