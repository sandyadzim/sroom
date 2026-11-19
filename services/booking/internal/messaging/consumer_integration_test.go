package messaging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"booking/internal/payment"

	amqp "github.com/rabbitmq/amqp091-go"
)

type brokerTestHandler struct{ processed chan struct{} }

func (h brokerTestHandler) Process(context.Context, []byte) error {
	h.processed <- struct{}{}
	return payment.ErrInvalidEvent
}

func TestRabbitMQConsumerAndDLQ(t *testing.T) {
	url := os.Getenv("BOOKING_RABBITMQ_TEST_URL")
	if url == "" {
		t.Skip("set BOOKING_RABBITMQ_TEST_URL to a dedicated sroom-test vhost")
	}
	uri, err := amqp.ParseURI(url)
	if err != nil || uri.Vhost != "sroom-test" {
		t.Fatal("consumer test requires the dedicated sroom-test vhost")
	}
	connection, err := amqp.DialConfig(url, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		t.Fatal("could not connect test broker")
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		t.Fatal("could not open test channel")
	}
	if err := declareTopology(channel); err != nil {
		t.Fatal("could not declare test topology")
	}
	defer func() {
		for _, name := range []string{Queue, DeadQueue} {
			if _, err := channel.QueueDelete(name, false, false, false); err != nil {
				t.Error("could not clean test queue")
			}
		}
		for _, name := range []string{Exchange, DeadExchange} {
			if err := channel.ExchangeDelete(name, false, false); err != nil {
				t.Error("could not clean test exchange")
			}
		}
	}()
	for _, name := range []string{Queue, DeadQueue} {
		if _, err := channel.QueuePurge(name, false); err != nil {
			t.Fatal("could not reset test queue")
		}
	}
	handler := brokerTestHandler{processed: make(chan struct{}, 1)}
	consumer := NewConsumer(url, handler, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); consumer.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(6 * time.Second):
			t.Error("consumer did not stop")
		}
	}()
	delivery := validDelivery(t)
	if err := channel.PublishWithContext(t.Context(), Exchange, payment.PaidEventType, true, false, amqp.Publishing{MessageId: delivery.MessageId, Type: delivery.Type, CorrelationId: delivery.CorrelationId, ContentType: delivery.ContentType, DeliveryMode: amqp.Persistent, Body: delivery.Body}); err != nil {
		t.Fatal("could not send synthetic test event")
	}
	select {
	case <-handler.processed:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer did not receive event")
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		message, ok, err := channel.Get(DeadQueue, false)
		if err != nil {
			t.Fatal("could not read test DLQ")
		}
		if ok {
			if message.MessageId != delivery.MessageId || message.Headers["error_code"] != "CONFIRMATION_REJECTED" || message.DeliveryMode != amqp.Persistent {
				t.Fatal("DLQ metadata incorrect")
			}
			if err := message.Ack(false); err != nil {
				t.Fatal(err)
			}
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("test cancelled")
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("rejected event was not retained in DLQ")
}
