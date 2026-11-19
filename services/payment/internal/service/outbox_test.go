package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"payment/internal/messaging"
	"payment/internal/model"
	"payment/internal/repository"

	"gorm.io/gorm"
)

type testEventPublisher struct {
	connectError error
	publishError error
	events       []model.OutboxEvent
	connects     int
	closes       int
	published    chan struct{}
}

func (p *testEventPublisher) Connect(context.Context) error { p.connects++; return p.connectError }
func (p *testEventPublisher) Publish(ctx context.Context, event model.OutboxEvent) error {
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

func outboxFixture(t *testing.T) (*gorm.DB, model.OutboxEvent) {
	t.Helper()
	// Relay polls every pending event: do not share the ordinary suite's database.
	dsn := os.Getenv("PAYMENT_OUTBOX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYMENT_OUTBOX_TEST_DATABASE_URL to a separate, migrated Payment test database")
	}
	if dsn == os.Getenv("PAYMENT_TEST_DATABASE_URL") {
		t.Fatal("outbox and ordinary suites must use separate databases")
	}
	t.Setenv("PAYMENT_TEST_DATABASE_URL", dsn)
	webhook, db, _, attempt, notification := webhookFixture(t)
	if err := webhook.Process(t.Context(), notification, notificationHash(notification), "outbox-test"); err != nil {
		t.Fatal(err)
	}
	var event model.OutboxEvent
	if err := db.Table("payment.outbox_events").Where("aggregate_id = ?", attempt.ID).Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	return db, event
}

func inspectOutbox(t *testing.T, db *gorm.DB, id string) (bool, int, *string) {
	t.Helper()
	var row struct {
		PublishedAt     *time.Time
		PublishAttempts int
		LastErrorCode   *string
	}
	if err := db.Table("payment.outbox_events").Where("id = ?", id).Take(&row).Error; err != nil {
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
			relay := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
				if err := db.Exec("UPDATE payment.outbox_events SET envelope = '{}' WHERE id = ?", event.ID).Error; err != nil {
					t.Fatal(err)
				}
			} else if err := db.Exec("UPDATE payment.outbox_events SET event_type = 'booking.demo_confirmed.v1' WHERE id = ?", event.ID).Error; err != nil {
				t.Fatal(err)
			}
			publisher := &testEventPublisher{}
			relay := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
	if err := db.Exec(`CREATE FUNCTION payment.test_reject_publication() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic publication failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP TRIGGER IF EXISTS test_reject_publication ON payment.outbox_events; DROP FUNCTION IF EXISTS payment.test_reject_publication()").Error; err != nil {
			t.Error("could not clean test trigger")
		}
	})
	if err := db.Exec(`CREATE TRIGGER test_reject_publication BEFORE UPDATE OF published_at ON payment.outbox_events FOR EACH ROW EXECUTE FUNCTION payment.test_reject_publication()`).Error; err != nil {
		t.Fatal(err)
	}
	publisher := &testEventPublisher{}
	relay := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := relay.ProcessBatch(t.Context()); !errors.Is(err, repository.ErrUnavailable) {
		t.Fatalf("recording error=%v", err)
	}
	published, _, _ := inspectOutbox(t, db, event.ID)
	if published || len(publisher.events) != 1 {
		t.Fatal("failed marking discarded the committed event")
	}
	if err := db.Exec("DROP TRIGGER test_reject_publication ON payment.outbox_events").Error; err != nil {
		t.Fatal(err)
	}
	// A new relay models restart after the first delivery reached the broker.
	restarted := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err := restarted.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	published, attempts, _ := inspectOutbox(t, db, event.ID)
	if !published || attempts != 2 || len(publisher.events) != 2 || publisher.events[0].ID != publisher.events[1].ID {
		t.Fatal("restart did not safely replay the same event")
	}
}

func TestOutboxBatchPostgres(t *testing.T) {
	db, original := outboxFixture(t)
	attempt := reloadAttempt(t, db, original.AggregateID)
	for range 20 {
		event := syntheticOutboxEvent(t)
		if err := db.Exec(`INSERT INTO payment.payment_attempts (id, booking_id, customer_id, provider_order_id, amount_idr, hold_expires_at) VALUES (?, ?, ?, ?, 150000, ?)`, event.AggregateID, event.AggregateID, attempt.CustomerID, "synthetic-"+event.AggregateID, attempt.HoldExpiresAt).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Exec(`INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (?, ?, ?, ?, ?)`, event.ID, event.AggregateID, event.EventType, string(event.Envelope), event.OccurredAt).Error; err != nil {
			t.Fatal(err)
		}
	}
	publisher := &testEventPublisher{}
	relay := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
	relay := NewOutboxRelay(repository.NewPaymentRepository(db), publisher, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
