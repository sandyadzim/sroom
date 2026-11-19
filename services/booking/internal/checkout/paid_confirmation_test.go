package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"booking/internal/payment"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type confirmationReader struct {
	data  payment.Confirmation
	err   error
	mu    sync.Mutex
	calls int
}

func (r *confirmationReader) Confirmation(context.Context, string, string) (payment.Confirmation, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return r.data, r.err
}

type confirmationFixture struct {
	service *PaidConfirmationService
	db      *gorm.DB
	reader  *confirmationReader
	event   payment.PaidEvent
	now     time.Time
}

func paidFixture(t *testing.T) confirmationFixture {
	t.Helper()
	dsn := os.Getenv("BOOKING_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BOOKING_TEST_DATABASE_URL to a dedicated migrated Booking test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("could not open test database")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	bookingID, customerID, paymentID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	at := time.Date(2030, 1, 10, 2, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, query := range []string{
			"DELETE FROM booking.inbox_events WHERE booking_id = ?",
			"DELETE FROM booking.outbox_events WHERE aggregate_id = ?",
			"DELETE FROM booking.booking_status_history WHERE booking_id = ?",
			"DELETE FROM booking.allocations WHERE booking_id = ?",
			"DELETE FROM booking.bookings WHERE id = ?",
		} {
			if err := db.WithContext(ctx).Exec(query, bookingID).Error; err != nil {
				t.Error("could not clean confirmation fixture")
			}
		}
	})
	if err := db.Exec(`INSERT INTO booking.bookings
  (id,customer_id,variant_id,status,pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at)
  VALUES (?, ?, ?, 'pending_payment','2030-01-11T10:00:00+08:00','2030-01-12T12:00:00+08:00','2030-01-11','2030-01-12','Synthetic car',150000,2,300000,'rental-terms-v1',?,?)`, bookingID, customerID, uuid.NewString(), at, at.Add(15*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO booking.allocations (booking_id,unit_id,occupied_period)
  VALUES (?, ?, tstzrange('2030-01-11T10:00:00+08:00','2030-01-12T12:00:00+08:00','[)'))`, bookingID, uuid.NewString()).Error; err != nil {
		t.Fatal(err)
	}
	verified := at.Add(time.Minute)
	reader := &confirmationReader{data: payment.Confirmation{PaymentID: paymentID, BookingID: bookingID, CustomerID: customerID, ProviderOrderID: "sroom-" + paymentID,
		AmountIDR: 300000, Currency: "IDR", Status: "paid", HoldExpiresAt: at.Add(15 * time.Minute), VerifiedAt: &verified}}
	event := payment.PaidEvent{EventID: uuid.NewString(), EventType: payment.PaidEventType, SchemaVersion: 1, OccurredAt: verified, AggregateID: paymentID, CorrelationID: "confirmation-test", CausationID: json.RawMessage("null"),
		Payload: payment.PaidPayload{PaymentID: paymentID, BookingID: bookingID, ProviderOrderID: reader.data.ProviderOrderID, AmountIDR: 300000, Currency: "IDR", VerifiedAt: verified, ProviderPaidAt: json.RawMessage("null")}}
	service := NewPaidConfirmationService(NewRepository(db), reader)
	service.now = func() time.Time { return at.Add(2 * time.Minute) }
	return confirmationFixture{service: service, db: db, reader: reader, event: event, now: at}
}

func eventBody(t *testing.T, event payment.PaidEvent) []byte {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func countRows(t *testing.T, f confirmationFixture, table, field string) int64 {
	t.Helper()
	var count int64
	if err := f.db.Table("booking."+table).Where(field+" = ?", f.event.Payload.BookingID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}
func bookingStatus(t *testing.T, f confirmationFixture) string {
	t.Helper()
	var status string
	if err := f.db.Raw("SELECT status FROM booking.bookings WHERE id = ?", f.event.Payload.BookingID).Scan(&status).Error; err != nil {
		t.Fatal(err)
	}
	return status
}

func TestVerifiedPaymentConfirmsPostgres(t *testing.T) {
	f := paidFixture(t)
	// Event arrival after the deadline still uses the persisted early verification proof.
	f.service.now = func() time.Time { return f.reader.data.HoldExpiresAt.Add(time.Hour) }
	body := eventBody(t, f.event)
	if err := f.service.Process(t.Context(), body); err != nil {
		t.Fatal(err)
	}
	if bookingStatus(t, f) != "confirmed" || countRows(t, f, "booking_status_history", "booking_id") != 1 || countRows(t, f, "outbox_events", "aggregate_id") != 1 || countRows(t, f, "inbox_events", "booking_id") != 1 {
		t.Fatal("confirmation did not commit all effects")
	}
	var blocking bool
	if err := f.db.Raw("SELECT blocks_inventory FROM booking.allocations WHERE booking_id = ?", f.event.Payload.BookingID).Scan(&blocking).Error; err != nil {
		t.Fatal(err)
	}
	if !blocking {
		t.Fatal("confirmation released inventory")
	}
	var outbox struct {
		EventType string
		Envelope  string
	}
	if err := f.db.Raw("SELECT event_type,envelope::text FROM booking.outbox_events WHERE aggregate_id = ?", f.event.Payload.BookingID).Scan(&outbox).Error; err != nil {
		t.Fatal(err)
	}
	if outbox.EventType != "booking.confirmed.v1" {
		t.Fatal("real confirmation emitted a demo event")
	}
	var recorded struct {
		CausationID string `json:"causation_id"`
		Payload     struct {
			BookingID   string    `json:"booking_id"`
			PaymentID   string    `json:"payment_id"`
			CustomerID  string    `json:"customer_id"`
			TotalIDR    int64     `json:"total_idr"`
			Currency    string    `json:"currency"`
			ConfirmedAt time.Time `json:"confirmed_at"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(outbox.Envelope), &recorded); err != nil {
		t.Fatal(err)
	}
	if recorded.CausationID != f.event.EventID || recorded.Payload.BookingID != f.event.Payload.BookingID ||
		recorded.Payload.PaymentID != f.reader.data.PaymentID || recorded.Payload.CustomerID != f.reader.data.CustomerID ||
		recorded.Payload.TotalIDR != 300000 || recorded.Payload.Currency != "IDR" || recorded.Payload.ConfirmedAt.IsZero() {
		t.Fatal("confirmed event lost the original references or price snapshot")
	}
	// A committed inbox lets redelivery succeed even when Payment is temporarily down.
	f.reader.err = payment.ErrUnavailable
	if err := f.service.Process(t.Context(), body); err != nil {
		t.Fatal(err)
	}
	if f.reader.calls != 1 {
		t.Fatal("duplicate unnecessarily depended on Payment")
	}
	f.reader.err = nil
	f.event.EventID = uuid.NewString()
	if err := f.service.Process(t.Context(), eventBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	if countRows(t, f, "outbox_events", "aggregate_id") != 1 || countRows(t, f, "booking_status_history", "booking_id") != 1 || countRows(t, f, "inbox_events", "booking_id") != 2 {
		t.Fatal("different event ID repeated business effects")
	}
}

func TestVerifiedPaymentReviewPostgres(t *testing.T) {
	for _, name := range []string{"at deadline", "after deadline", "review flag", "expired", "released", "missing allocation", "wrong period", "snapshot mismatch"} {
		t.Run(name, func(t *testing.T) {
			f := paidFixture(t)
			switch name {
			case "at deadline", "after deadline":
				verified := f.reader.data.HoldExpiresAt
				if name == "after deadline" {
					verified = verified.Add(time.Second)
				}
				f.reader.data.VerifiedAt = &verified
				f.event.Payload.VerifiedAt = verified
				f.event.OccurredAt = verified
			case "review flag":
				f.reader.data.ReviewRequired = true
			case "expired":
				if err := f.db.Exec("UPDATE booking.bookings SET status = 'expired' WHERE id = ?", f.event.Payload.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "released":
				if err := f.db.Exec("UPDATE booking.allocations SET blocks_inventory = FALSE,released_at = NOW() WHERE booking_id = ?", f.event.Payload.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "missing allocation":
				if err := f.db.Exec("DELETE FROM booking.allocations WHERE booking_id = ?", f.event.Payload.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "wrong period":
				if err := f.db.Exec("UPDATE booking.allocations SET occupied_period = tstzrange('2030-01-11T11:00:00+08:00','2030-01-12T12:00:00+08:00','[)') WHERE booking_id = ?", f.event.Payload.BookingID).Error; err != nil {
					t.Fatal(err)
				}
			case "snapshot mismatch":
				f.reader.data.HoldExpiresAt = f.reader.data.HoldExpiresAt.Add(time.Minute)
			}
			if err := f.service.Process(t.Context(), eventBody(t, f.event)); err != nil {
				t.Fatal(err)
			}
			if bookingStatus(t, f) != "payment_review" || countRows(t, f, "outbox_events", "aggregate_id") != 0 || countRows(t, f, "inbox_events", "booking_id") != 1 {
				t.Fatal("ineligible paid was confirmed or lost")
			}
			if name == "released" {
				var blocking bool
				if err := f.db.Raw("SELECT blocks_inventory FROM booking.allocations WHERE booking_id = ?", f.event.Payload.BookingID).Scan(&blocking).Error; err != nil || blocking {
					t.Fatal("review reclaimed released inventory")
				}
			}
		})
	}
}

func TestVerifiedPaymentRejectsAndRollbackPostgres(t *testing.T) {
	for _, name := range []string{"mismatch", "unavailable", "not paid", "confirmed elsewhere", "rollback"} {
		t.Run(name, func(t *testing.T) {
			f := paidFixture(t)
			var want error
			switch name {
			case "mismatch":
				f.reader.data.AmountIDR++
				want = payment.ErrInvalidEvent
			case "unavailable":
				f.reader.err = payment.ErrUnavailable
				want = payment.ErrUnavailable
			case "not paid":
				f.reader.data.Status = "pending"
				want = payment.ErrInvalidEvent
			case "confirmed elsewhere":
				if err := f.db.Exec("UPDATE booking.bookings SET status = 'confirmed' WHERE id = ?", f.event.Payload.BookingID).Error; err != nil {
					t.Fatal(err)
				}
				want = ErrConfirmationConflict
			case "rollback":
				if err := f.db.Exec(`INSERT INTO booking.outbox_events (id,aggregate_id,event_type,envelope,occurred_at) VALUES (?, ?, 'booking.confirmed.v1','{}',?)`, uuid.NewString(), f.event.Payload.BookingID, f.now).Error; err != nil {
					t.Fatal(err)
				}
				want = ErrDatabaseUnavailable
			}
			if err := f.service.Process(t.Context(), eventBody(t, f.event)); !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			expected := "pending_payment"
			if name == "confirmed elsewhere" {
				expected = "confirmed"
			}
			if bookingStatus(t, f) != expected || countRows(t, f, "booking_status_history", "booking_id") != 0 || countRows(t, f, "inbox_events", "booking_id") != 0 {
				t.Fatal("rejection or rollback left partial state")
			}
		})
	}
}

func TestVerifiedPaymentConcurrentAndEventConflictPostgres(t *testing.T) {
	f := paidFixture(t)
	body := eventBody(t, f.event)
	var wg sync.WaitGroup
	results := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- f.service.Process(t.Context(), body) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if countRows(t, f, "outbox_events", "aggregate_id") != 1 || countRows(t, f, "booking_status_history", "booking_id") != 1 {
		t.Fatal("concurrent duplicate repeated effects")
	}
	f.event.Payload.AmountIDR++
	if err := f.service.Process(t.Context(), eventBody(t, f.event)); !errors.Is(err, ErrConfirmationConflict) {
		t.Fatalf("changed payload reused event ID: %v", err)
	}
}
