package messaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"payment/internal/model"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Only a dedicated test vhost may be purged by this transport smoke test.
func TestRabbitMQTransport(t *testing.T) {
	url := os.Getenv("PAYMENT_RABBITMQ_TEST_URL")
	if url == "" {
		t.Skip("set PAYMENT_RABBITMQ_TEST_URL to a dedicated RabbitMQ sroom-test vhost")
	}
	uri, err := amqp.ParseURI(url)
	if err != nil || uri.Vhost != "sroom-test" {
		t.Fatal("RabbitMQ test URL must use the dedicated sroom-test vhost")
	}
	publisher := NewRabbitPublisher(url)
	if err := publisher.Connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := publisher.Connect(t.Context()); err != nil {
		t.Fatal("topology redeclaration failed")
	}
	if _, err := publisher.channel.QueuePurge(BookingQueue, false); err != nil {
		t.Fatal("could not clear dedicated test queue")
	}
	defer func() {
		if publisher.channel != nil {
			if _, err := publisher.channel.QueueDelete(BookingQueue, false, false, false); err != nil {
				t.Error("could not clean test queue")
			}
			if err := publisher.channel.ExchangeDelete(Exchange, false, false); err != nil {
				t.Error("could not clean test exchange")
			}
		}
	}()
	event := model.OutboxEvent{ID: uuid.NewString(), AggregateID: uuid.NewString(), EventType: PaidEvent, OccurredAt: time.Now().UTC().Truncate(time.Second), CorrelationID: "rabbitmq-transport-test"}
	event.Envelope, err = json.Marshal(map[string]any{"event_id": event.ID, "aggregate_id": event.AggregateID, "event_type": PaidEvent, "schema_version": 1,
		"occurred_at": event.OccurredAt, "correlation_id": event.CorrelationID, "causation_id": nil,
		"payload": map[string]any{"payment_id": event.AggregateID, "booking_id": uuid.NewString(), "provider_order_id": "synthetic-order", "amount_idr": 150000, "currency": "IDR", "verified_at": event.OccurredAt, "provider_paid_at": nil}})
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(t.Context(), event); err != nil {
		t.Fatal(err)
	}
	// Client restart must leave the persistent event queued; no business consumer exists.
	if err := publisher.Close(); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	delivery, ok, err := publisher.channel.Get(BookingQueue, false)
	if err != nil || !ok {
		t.Fatal("confirmed message was not retained")
	}
	if delivery.MessageId != event.ID || delivery.CorrelationId != event.CorrelationID || delivery.Type != PaidEvent ||
		delivery.ContentType != "application/json" || delivery.DeliveryMode != amqp.Persistent || delivery.RoutingKey != PaidEvent || !bytes.Equal(delivery.Body, event.Envelope) {
		t.Fatal("message metadata/body changed")
	}
	if err := delivery.Ack(false); err != nil {
		t.Fatal(err)
	}
	if err := publisher.channel.QueueUnbind(BookingQueue, PaidEvent, Exchange, nil); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(t.Context(), event); !errors.Is(err, ErrUnroutable) {
		t.Fatalf("unroutable result=%v", err)
	}
}
