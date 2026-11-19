package messaging

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"booking/internal/checkout"
	"booking/internal/payment"

	amqp "github.com/rabbitmq/amqp091-go"
)

const Exchange = "sroom.domain.v1"
const Queue = "booking.payment-events.v1"
const DeadExchange = "sroom.dlx.v1"
const DeadQueue = "booking.payment-events.dlq.v1"

var ErrBroker = errors.New("booking event broker unavailable")

type PaidHandler interface {
	Process(context.Context, []byte) error
}

type Consumer struct {
	url        string
	handler    PaidHandler
	log        *slog.Logger
	retryDelay time.Duration
}

func NewConsumer(url string, handler PaidHandler, log *slog.Logger) *Consumer {
	return &Consumer{url: url, handler: handler, log: log, retryDelay: time.Second}
}

func (c *Consumer) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := c.consume(ctx); err != nil && ctx.Err() == nil {
			c.log.Warn("booking event connection unavailable", "operation", "events.consume")
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func declareTopology(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(Queue, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		return err
	}
	if err := channel.QueueBind(Queue, payment.PaidEventType, Exchange, false, nil); err != nil {
		return err
	}
	if err := channel.ExchangeDeclare(DeadExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(DeadQueue, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		return err
	}
	if err := channel.QueueBind(DeadQueue, Queue, DeadExchange, false, nil); err != nil {
		return err
	}
	if err := channel.Qos(1, 0, false); err != nil {
		return err
	}
	return channel.Confirm(false)
}

func (c *Consumer) consume(ctx context.Context) error {
	setup, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var socket net.Conn
	var stopSetup func() bool
	connection, err := amqp.DialConfig(c.url, amqp.Config{Heartbeat: 10 * time.Second, Dial: func(network, address string) (net.Conn, error) {
		connection, err := (&net.Dialer{}).DialContext(setup, network, address)
		if err != nil {
			return nil, err
		}
		socket = connection
		stopSetup = context.AfterFunc(setup, func() { _ = connection.Close() })
		return connection, nil
	}})
	defer func() {
		if stopSetup != nil {
			stopSetup()
		}
	}()
	if err != nil {
		if socket != nil {
			_ = socket.Close()
		}
		return ErrBroker
	}
	defer func() {
		// Closing the socket also unblocks any pending AMQP RPC on shutdown/error.
		_ = socket.Close()
		if err := connection.CloseDeadline(time.Now().Add(time.Second)); err != nil && !errors.Is(err, amqp.ErrClosed) {
			c.log.Warn("booking event close failed", "operation", "events.close")
		}
	}()
	channel, err := connection.Channel()
	if err != nil {
		return ErrBroker
	}
	if err := declareTopology(channel); err != nil {
		return ErrBroker
	}
	returns := channel.NotifyReturn(make(chan amqp.Return, 1))
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	deliveries, err := channel.ConsumeWithContext(ctx, Queue, "", false, false, false, false, nil)
	if err != nil {
		return ErrBroker
	}
	if setup.Err() != nil {
		return ErrBroker
	}
	stopSetup()
	cancel()
	stopLifetime := context.AfterFunc(ctx, func() { _ = socket.Close() })
	defer stopLifetime()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-deliveries:
			if !ok {
				return ErrBroker
			}
			err := c.handleDelivery(ctx, delivery, func(ctx context.Context, delivery amqp.Delivery, code string) error {
				return publishDeadLetter(ctx, channel, socket, confirms, returns, delivery, code)
			})
			if err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, delivery amqp.Delivery, deadLetter func(context.Context, amqp.Delivery, string) error) error {
	event, err := payment.DecodePaidEvent(delivery.Body)
	if err == nil && (delivery.ContentType != "application/json" || delivery.Type != payment.PaidEventType || delivery.RoutingKey != payment.PaidEventType || delivery.MessageId != event.EventID || delivery.CorrelationId != event.CorrelationID) {
		err = payment.ErrInvalidEvent
	}
	code := "INVALID_EVENT"
	if err == nil {
		for attempt := 0; attempt < 3; attempt++ {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			err = c.handler.Process(ctx, delivery.Body)
			if err == nil {
				return delivery.Ack(false)
			}
			if errors.Is(err, payment.ErrInvalidEvent) || errors.Is(err, checkout.ErrConfirmationConflict) || errors.Is(err, checkout.ErrBookingNotFound) {
				code = "CONFIRMATION_REJECTED"
				break
			}
			code = "PROCESSING_UNAVAILABLE"
			if attempt < 2 {
				timer := time.NewTimer(c.retryDelay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
				}
			}
		}
	}
	// No ACK on failure unless the durable DLQ publish is confirmed first.
	if err := deadLetter(ctx, delivery, code); err != nil {
		return err
	}
	c.log.Warn("booking event retained in DLQ", "operation", "events.dead_letter", "error_code", code)
	return delivery.Ack(false)
}

func publishDeadLetter(ctx context.Context, channel *amqp.Channel, socket net.Conn, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, delivery amqp.Delivery, code string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = socket.Close() })
	defer stop()
	sequence := channel.GetNextPublishSeqNo()
	err := channel.PublishWithContext(ctx, DeadExchange, Queue, true, false, amqp.Publishing{ContentType: delivery.ContentType, DeliveryMode: amqp.Persistent,
		MessageId: delivery.MessageId, Type: delivery.Type, CorrelationId: delivery.CorrelationId, Body: delivery.Body, Headers: amqp.Table{"error_code": code}})
	if err != nil {
		return ErrBroker
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-returns:
		return ErrBroker
	case confirm, ok := <-confirms:
		if !ok || !confirm.Ack || confirm.DeliveryTag != sequence {
			return ErrBroker
		}
		select {
		case <-returns:
			return ErrBroker
		default:
		}
		return nil
	}
}
