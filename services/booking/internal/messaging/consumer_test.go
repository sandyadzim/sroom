package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"booking/internal/payment"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type testPaidHandler struct {
	calls     int
	failUntil int
	err       error
}

func (h *testPaidHandler) Process(context.Context, []byte) error {
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
	eventID, paymentID := uuid.NewString(), uuid.NewString()
	at := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	body, err := json.Marshal(map[string]any{"event_id": eventID, "event_type": payment.PaidEventType, "schema_version": 1, "occurred_at": at,
		"aggregate_id": paymentID, "correlation_id": "consumer-test", "causation_id": nil, "payload": map[string]any{"payment_id": paymentID, "booking_id": uuid.NewString(), "provider_order_id": "synthetic-order", "amount_idr": 150000, "currency": "IDR", "verified_at": at, "provider_paid_at": nil}})
	if err != nil {
		t.Fatal(err)
	}
	return amqp.Delivery{MessageId: eventID, Type: payment.PaidEventType, RoutingKey: payment.PaidEventType, CorrelationId: "consumer-test", ContentType: "application/json", Body: body, Acknowledger: &testAcknowledger{}}
}

func TestConsumerCommitRetryAndDLQ(t *testing.T) {
	for _, name := range []string{"committed", "retry succeeds", "exhausted", "invalid", "permanent", "dlq fails", "cancelled"} {
		t.Run(name, func(t *testing.T) {
			delivery := validDelivery(t)
			handler := &testPaidHandler{}
			consumer := NewConsumer("", handler, slog.New(slog.NewJSONHandler(io.Discard, nil)))
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
				handler.err = payment.ErrInvalidEvent
			case "cancelled":
				cancel()
			}
			if name == "dlq fails" {
				deadLetterError = ErrBroker
			}
			err := consumer.handleDelivery(ctx, delivery, func(_ context.Context, _ amqp.Delivery, _ string) error {
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
			if (name == "invalid" || name == "permanent" || name == "exhausted") && deadLetters != 1 {
				t.Fatal("poison event was not retained")
			}
			if name == "invalid" && handler.calls != 0 {
				t.Fatal("invalid event reached business handler")
			}
			if name == "permanent" && handler.calls != 1 {
				t.Fatal("permanent rejection retried")
			}
		})
	}
}
