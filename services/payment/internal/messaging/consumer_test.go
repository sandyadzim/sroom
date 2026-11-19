package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type testInvoiceHandler struct {
	calls     int
	failUntil int
	err       error
}

func (h *testInvoiceHandler) Process(context.Context, []byte) error {
	h.calls++
	if h.calls <= h.failUntil {
		return h.err
	}
	return nil
}

type testAcknowledger struct{ acked int }

func (a *testAcknowledger) Ack(uint64, bool) error        { a.acked++; return nil }
func (a *testAcknowledger) Nack(uint64, bool, bool) error { return errors.New("unexpected nack") }
func (a *testAcknowledger) Reject(uint64, bool) error     { return errors.New("unexpected reject") }

func validDelivery(t *testing.T) amqp.Delivery {
	t.Helper()
	eventID, bookingID := uuid.NewString(), uuid.NewString()
	at := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	body, err := json.Marshal(model.ConfirmedEvent{EventID: eventID, EventType: model.ConfirmedEventType, SchemaVersion: 1, OccurredAt: at, AggregateID: bookingID, CorrelationID: "consumer-test", CausationID: uuid.NewString(), Payload: model.ConfirmedPayload{BookingID: bookingID, PaymentID: uuid.NewString(), CustomerID: uuid.NewString(), ConfirmedAt: at, TotalIDR: 150000, Currency: "IDR"}})
	if err != nil {
		t.Fatal(err)
	}
	return amqp.Delivery{MessageId: eventID, Type: model.ConfirmedEventType, RoutingKey: model.ConfirmedEventType, CorrelationId: "consumer-test", ContentType: "application/json", Body: body, Acknowledger: &testAcknowledger{}}
}

func TestInvoiceConsumerCommitRetryAndDLQ(t *testing.T) {
	for _, name := range []string{"committed", "retry succeeds", "exhausted", "invalid", "permanent", "legacy", "dlq fails", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			delivery := validDelivery(t)
			handler := &testInvoiceHandler{}
			consumer := NewInvoiceConsumer("", handler, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			consumer.retryDelay = 0
			deadLetters := 0
			var deadLetterError error
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			switch name {
			case "retry succeeds":
				handler.failUntil = 2
				handler.err = errors.New("temporary")
			case "exhausted", "dlq fails":
				handler.failUntil = 3
				handler.err = errors.New("temporary")
			case "invalid":
				delivery.Body = []byte(`{}`)
			case "permanent":
				handler.failUntil = 3
				handler.err = model.ErrInvalidConfirmation
			case "legacy":
				handler.failUntil = 3
				handler.err = repository.ErrInvoiceProfileMissing
			case "cancelled":
				cancel()
			}
			if name == "dlq fails" {
				deadLetterError = ErrInvoiceBroker
			}
			err := consumer.handleDelivery(ctx, delivery, func(_ context.Context, _ amqp.Delivery, code string) error {
				if name == "legacy" && code != "CUSTOMER_SNAPSHOT_MISSING" {
					t.Error("missing billing profile lost its safe reason")
				}
				if delivery.Acknowledger.(*testAcknowledger).acked != 0 {
					t.Error("ACK occurred before DLQ confirmation")
				}
				deadLetters++
				return deadLetterError
			})
			acked := delivery.Acknowledger.(*testAcknowledger).acked
			if name == "dlq fails" || name == "cancelled" {
				if err == nil || acked != 0 {
					t.Fatal("failed/cancelled work was acknowledged")
				}
				return
			}
			if err != nil || acked != 1 {
				t.Fatalf("ack=%d error=%v", acked, err)
			}
			if (name == "retry succeeds" || name == "exhausted") && handler.calls != 3 {
				t.Fatal("retry count incorrect")
			}
			if (name == "invalid" || (name == "permanent" || name == "legacy") || name == "exhausted") && deadLetters != 1 {
				t.Fatal("poison event was not retained")
			}
			if name == "invalid" && handler.calls != 0 {
				t.Fatal("invalid event reached business handler")
			}
			if (name == "permanent" || name == "legacy") && handler.calls != 1 {
				t.Fatal("permanent rejection retried")
			}
		})
	}
}
