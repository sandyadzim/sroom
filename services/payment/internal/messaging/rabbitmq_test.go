package messaging

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestPublisherConfirm(t *testing.T) {
	for _, name := range []string{"ack", "nack", "unroutable", "return before ack", "closed confirms", "closed returns", "wrong sequence", "timeout"} {
		t.Run(name, func(t *testing.T) {
			confirms := make(chan amqp.Confirmation, 1)
			returns := make(chan amqp.Return, 1)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
			defer cancel()
			var want error
			switch name {
			case "ack":
				confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}
			case "nack":
				confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: false}
				want = ErrPublishNack
			case "unroutable":
				returns <- amqp.Return{}
				want = ErrUnroutable
			case "return before ack":
				returns <- amqp.Return{}
				confirms <- amqp.Confirmation{DeliveryTag: 1, Ack: true}
				want = ErrUnroutable
			case "closed confirms":
				close(confirms)
				want = ErrPublishFailed
			case "closed returns":
				close(returns)
				want = ErrBrokerUnavailable
			case "wrong sequence":
				confirms <- amqp.Confirmation{DeliveryTag: 2, Ack: true}
				want = ErrPublishFailed
			case "timeout":
				want = context.DeadlineExceeded
			}
			if err := waitForConfirmation(ctx, 1, confirms, returns); !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
		})
	}
}

func TestConnectCancellationAndSafeError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		_, _ = io.Copy(io.Discard, connection) // Deliberately never answer the AMQP handshake.
	}()
	publisher := NewRabbitPublisher("amqp://test-user:synthetic-password@" + listener.Addr().String() + "/test")
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err = publisher.Connect(ctx)
	if !errors.Is(err, ErrBrokerUnavailable) || strings.Contains(err.Error(), "synthetic-password") || time.Since(start) > time.Second {
		t.Fatal("connection was not bounded or safely reported")
	}
	if err := publisher.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("canceled handshake left socket open")
	}
}
