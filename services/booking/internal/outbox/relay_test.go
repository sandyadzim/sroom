package outbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"booking/internal/checkout"
	"booking/internal/messaging"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testEventPublisher struct {
	connectError error
	publishError error
	events       []checkout.OutboxEvent
	connects     int
	closes       int
	published    chan struct{}
}

func (p *testEventPublisher) Connect(context.Context) error { p.connects++; return p.connectError }
func (p *testEventPublisher) Publish(ctx context.Context, event checkout.OutboxEvent) error {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
		return errors.New("missing publication deadline")
	}
	p.events = append(p.events, event)
	if p.published != nil {
		p.published <- struct{}{}
	}
	return p.publishError
}
func (p *testEventPublisher) Close() error { p.closes++; return nil }

func outboxFixture(t *testing.T) (*gorm.DB, checkout.OutboxEvent) {
	t.Helper()
	dsn := os.Getenv("BOOKING_OUTBOX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BOOKING_OUTBOX_TEST_DATABASE_URL to a separate migrated Booking test database")
	}
	if dsn == os.Getenv("BOOKING_TEST_DATABASE_URL") {
		t.Fatal("outbox and ordinary suites must use separate databases")
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
	return db, seedConfirmedEvent(t, db)
}
func seedConfirmedEvent(t *testing.T, db *gorm.DB) checkout.OutboxEvent {
	t.Helper()
	bookingID, eventID := uuid.NewString(), uuid.NewString()
	at := time.Date(2030, 1, 1, 2, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		for _, query := range []string{"DELETE FROM booking.outbox_events WHERE aggregate_id = ?", "DELETE FROM booking.bookings WHERE id = ?"} {
			if err := db.Exec(query, bookingID).Error; err != nil {
				t.Error(err)
			}
		}
	})
	if err := db.Exec(`INSERT INTO booking.bookings (id,customer_id,variant_id,status,pickup_at,return_at,pickup_local_date,return_local_date,vehicle_label_snapshot,daily_rate_idr,rental_days,total_idr,terms_version,terms_accepted_at,hold_expires_at)
 VALUES (?, ?, ?, 'confirmed','2030-01-02T10:00:00+08:00','2030-01-02T12:00:00+08:00','2030-01-02','2030-01-02','Synthetic car',150000,1,150000,'rental-terms-v1',?,?)`, bookingID, uuid.NewString(), uuid.NewString(), at, at.Add(15*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"event_id": eventID, "event_type": "booking.confirmed.v1", "schema_version": 1, "occurred_at": at, "aggregate_id": bookingID, "correlation_id": "outbox-test", "causation_id": uuid.NewString(), "payload": map[string]any{"booking_id": bookingID, "payment_id": uuid.NewString(), "customer_id": uuid.NewString(), "confirmed_at": at, "total_idr": 150000, "currency": "IDR"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO booking.outbox_events (id,aggregate_id,event_type,envelope,occurred_at) VALUES (?, ?, 'booking.confirmed.v1',CAST(? AS JSONB),?)", eventID, bookingID, string(body), at).Error; err != nil {
		t.Fatal(err)
	}
	return checkout.OutboxEvent{ID: eventID, AggregateID: bookingID, EventType: "booking.confirmed.v1", Envelope: body, OccurredAt: at}
}

func inspectOutbox(t *testing.T, db *gorm.DB, id string) (bool, int, *string) {
	t.Helper()
	var row struct {
		PublishedAt     *time.Time
		PublishAttempts int
		LastErrorCode   *string
	}
	if err := db.Table("booking.outbox_events").Where("id = ?", id).Take(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row.PublishedAt != nil, row.PublishAttempts, row.LastErrorCode
}

func TestOutboxPublishAndRetryPostgres(t *testing.T) {
	for _, name := range []string{"success", "nack", "unroutable", "timeout", "ambiguous publish"} {
		t.Run(name, func(t *testing.T) {
			db, event := outboxFixture(t)
			publisher := &testEventPublisher{}
			switch name {
			case "nack":
				publisher.publishError = messaging.ErrPublishNack
			case "unroutable":
				publisher.publishError = messaging.ErrUnroutable
			case "timeout":
				publisher.publishError = context.DeadlineExceeded
			case "ambiguous publish":
				publisher.publishError = messaging.ErrPublishFailed
			}
			relay := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			err := relay.ProcessBatch(t.Context())
			published, attempts, code := inspectOutbox(t, db, event.ID)
			if name == "success" {
				if err != nil || !published || attempts != 1 || code != nil {
					t.Fatal("confirmed publication was not marked")
				}
			} else {
				if err == nil || published || attempts != 1 || code == nil || *code != publicationErrorCode(publisher.publishError) {
					t.Fatal("failed publication lost its pending state")
				}
				publisher.publishError = nil
				if err := relay.ProcessBatch(t.Context()); err != nil {
					t.Fatal(err)
				}
				if len(publisher.events) != 2 || publisher.events[0].ID != publisher.events[1].ID || !bytes.Equal(publisher.events[0].Envelope, publisher.events[1].Envelope) {
					t.Fatal("retry replaced the original event")
				}
			}
			previous := len(publisher.events)
			if err := relay.ProcessBatch(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(publisher.events) != previous || publisher.events[0].CorrelationID != "outbox-test" {
				t.Fatal("published event repeated or correlation lost")
			}
		})
	}
}

func TestOutboxInvalidAndUnsupportedPostgres(t *testing.T) {
	for _, name := range []string{"invalid", "demo"} {
		t.Run(name, func(t *testing.T) {
			db, event := outboxFixture(t)
			if name == "invalid" {
				if err := db.Exec("UPDATE booking.outbox_events SET envelope = '{}' WHERE id = ?", event.ID).Error; err != nil {
					t.Fatal(err)
				}
			} else if err := db.Exec("UPDATE booking.outbox_events SET event_type = 'booking.demo_confirmed.v1' WHERE id = ?", event.ID).Error; err != nil {
				t.Fatal(err)
			}
			publisher := &testEventPublisher{}
			relay := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			for range 2 {
				if err := relay.ProcessBatch(t.Context()); err != nil {
					t.Fatal(err)
				}
			}
			published, attempts, code := inspectOutbox(t, db, event.ID)
			if published || attempts != 0 || len(publisher.events) != 0 {
				t.Fatal("invalid or demo event was published")
			}
			if name == "invalid" && (code == nil || *code != "INVALID_EVENT") {
				t.Fatal("invalid event not quarantined")
			}
		})
	}
}

func TestOutboxCrashAfterConfirmPostgres(t *testing.T) {
	db, event := outboxFixture(t)
	if err := db.Exec(`CREATE FUNCTION booking.test_reject_publication() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic publication failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP TRIGGER IF EXISTS test_reject_publication ON booking.outbox_events; DROP FUNCTION IF EXISTS booking.test_reject_publication()").Error; err != nil {
			t.Error("could not clean test trigger")
		}
	})
	if err := db.Exec(`CREATE TRIGGER test_reject_publication BEFORE UPDATE OF published_at ON booking.outbox_events FOR EACH ROW EXECUTE FUNCTION booking.test_reject_publication()`).Error; err != nil {
		t.Fatal(err)
	}
	publisher := &testEventPublisher{}
	relay := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := relay.ProcessBatch(t.Context()); !errors.Is(err, checkout.ErrDatabaseUnavailable) {
		t.Fatalf("recording error=%v", err)
	}
	published, _, _ := inspectOutbox(t, db, event.ID)
	if published || len(publisher.events) != 1 {
		t.Fatal("failed marking discarded the committed event")
	}
	if err := db.Exec("DROP TRIGGER test_reject_publication ON booking.outbox_events").Error; err != nil {
		t.Fatal(err)
	}
	// A new relay models restart after the first delivery reached the broker.
	restarted := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := restarted.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	published, attempts, _ := inspectOutbox(t, db, event.ID)
	if !published || attempts != 2 || len(publisher.events) != 2 || publisher.events[0].ID != publisher.events[1].ID {
		t.Fatal("restart did not safely replay the same event")
	}
}

func TestOutboxBatchPostgres(t *testing.T) {
	db, _ := outboxFixture(t)
	for range 20 {
		seedConfirmedEvent(t, db)
	}
	publisher := &testEventPublisher{}
	relay := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := relay.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(publisher.events) != 20 {
		t.Fatalf("batch size=%d", len(publisher.events))
	}
	if err := relay.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(publisher.events) != 21 {
		t.Fatal("next batch did not continue")
	}
	for i := 1; i < len(publisher.events); i++ {
		previous, current := publisher.events[i-1], publisher.events[i]
		if previous.OccurredAt.After(current.OccurredAt) || (previous.OccurredAt.Equal(current.OccurredAt) && previous.ID > current.ID) {
			t.Fatal("outbox ordering is unstable")
		}
	}
}

func TestOutboxRunShutdown(t *testing.T) {
	publisher := &testEventPublisher{}
	relay := NewOutboxRelay(nil, publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	relay.Run(ctx)
	if publisher.connects != 0 || publisher.closes != 1 {
		t.Fatal("canceled relay performed work or failed to close")
	}
}

func TestOutboxRunAndBrokerRecoveryPostgres(t *testing.T) {
	db, event := outboxFixture(t)
	publisher := &testEventPublisher{connectError: messaging.ErrBrokerUnavailable}
	relay := NewOutboxRelay(checkout.NewRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := relay.ProcessBatch(t.Context()); !errors.Is(err, messaging.ErrBrokerUnavailable) {
		t.Fatal("broker failure not reported")
	}
	published, attempts, _ := inspectOutbox(t, db, event.ID)
	if published || attempts != 0 {
		t.Fatal("broker outage changed pending facts")
	}
	publisher.connectError = nil
	publisher.published = make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan struct{})
	go func() { defer close(finished); relay.Run(ctx) }()
	select {
	case <-publisher.published:
	case <-time.After(time.Second):
		t.Fatal("relay did not run at startup")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("relay did not stop")
	}
	if publisher.closes != 1 {
		t.Fatal("relay did not close broker connection")
	}
}
