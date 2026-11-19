package messaging

import (
	"context"
	"errors"
	"net"
	"time"

	"payment/internal/model"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "sroom.domain.v1"
const BookingQueue = "booking.payment-events.v1"
const PaidEvent = "payment.paid.v1"

var (
	ErrBrokerUnavailable = errors.New("message broker unavailable")
	ErrTopology          = errors.New("message broker topology mismatch or unavailable")
	ErrPublishFailed     = errors.New("event publication failed")
	ErrPublishNack       = errors.New("event publication rejected")
	ErrUnroutable        = errors.New("event has no destination queue")
)

// A single relay owns this connection and publishes one event at a time.
type RabbitPublisher struct {
	url        string
	connection *amqp.Connection
	channel    *amqp.Channel
	socket     net.Conn
	returns    <-chan amqp.Return
	confirms   <-chan amqp.Confirmation
}

func NewRabbitPublisher(url string) *RabbitPublisher { return &RabbitPublisher{url: url} }

func (p *RabbitPublisher) Connect(ctx context.Context) error {
	if p.connection != nil && !p.connection.IsClosed() && p.channel != nil && !p.channel.IsClosed() {
		return nil
	}
	if err := p.Close(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var stopCancellation func() bool
	// AMQP RPCs do not accept a context. Closing their socket interrupts blocked I/O.
	connection, err := amqp.DialConfig(p.url, amqp.Config{Heartbeat: 10 * time.Second, Dial: func(network, address string) (net.Conn, error) {
		socket, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		p.socket = socket
		stopCancellation = context.AfterFunc(ctx, func() { _ = socket.Close() })
		return socket, nil
	}})
	defer func() {
		if stopCancellation != nil {
			stopCancellation()
		}
	}()
	if err != nil {
		// The dial error may contain credentials. Return only an application-owned error.
		if p.socket != nil {
			_ = p.Close()
		}
		return ErrBrokerUnavailable
	}
	p.connection = connection
	p.channel, err = connection.Channel()
	if err == nil {
		err = p.channel.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil)
	}
	if err == nil {
		_, err = p.channel.QueueDeclare(BookingQueue, true, false, false, false, amqp.Table{"x-queue-type": "classic"})
	}
	if err == nil {
		err = p.channel.QueueBind(BookingQueue, PaidEvent, Exchange, false, nil)
	}
	if err == nil {
		err = p.channel.Confirm(false)
	}
	if err != nil {
		_ = p.Close()
		return ErrTopology
	}
	if ctx.Err() != nil {
		_ = p.Close()
		return ctx.Err()
	}
	p.returns = p.channel.NotifyReturn(make(chan amqp.Return, 1))
	p.confirms = p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	return nil
}

func (p *RabbitPublisher) Publish(ctx context.Context, event model.OutboxEvent) error {
	if p.connection == nil || p.channel == nil {
		return ErrBrokerUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	socket := p.socket
	// PublishWithContext only checks cancellation before its write starts.
	stopCancellation := context.AfterFunc(ctx, func() { _ = socket.Close() })
	defer stopCancellation()
	sequence := p.channel.GetNextPublishSeqNo()
	err := p.channel.PublishWithContext(ctx, Exchange, event.EventType, true, false, amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: event.ID,
		Type: event.EventType, CorrelationId: event.CorrelationID, Timestamp: event.OccurredAt.UTC(), Body: event.Envelope,
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrPublishFailed
	}
	return waitForConfirmation(ctx, sequence, p.confirms, p.returns)
}

func waitForConfirmation(ctx context.Context, sequence uint64, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case _, ok := <-returns:
		if !ok {
			return ErrBrokerUnavailable
		}
		return ErrUnroutable
	case confirmation, ok := <-confirms:
		if !ok || confirmation.DeliveryTag != sequence {
			return ErrPublishFailed
		}
		// RabbitMQ returns an unroutable mandatory message before its confirm.
		// Both notifications may already be buffered when the select chooses confirm.
		select {
		case _, ok := <-returns:
			if !ok {
				return ErrBrokerUnavailable
			}
			return ErrUnroutable
		default:
		}
		if !confirmation.Ack {
			return ErrPublishNack
		}
		return nil
	}
}

func (p *RabbitPublisher) Close() error {
	var closeError error
	if p.connection != nil && !p.connection.IsClosed() {
		closeError = p.connection.CloseDeadline(time.Now().Add(time.Second))
	}
	if p.socket != nil {
		if err := p.socket.Close(); err != nil && !errors.Is(err, net.ErrClosed) && closeError == nil {
			closeError = err
		}
	}
	p.connection, p.channel, p.socket = nil, nil, nil
	p.returns, p.confirms = nil, nil
	if closeError != nil && !errors.Is(closeError, amqp.ErrClosed) {
		return ErrBrokerUnavailable
	}
	return nil
}
