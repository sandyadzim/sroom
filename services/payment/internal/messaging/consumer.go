package messaging

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"payment/internal/booking"
	"payment/internal/model"
	"payment/internal/repository"

	amqp "github.com/rabbitmq/amqp091-go"
)

const InvoiceQueue = "payment.booking-events.v1"
const InvoiceDeadExchange = "sroom.dlx.v1"
const InvoiceDeadQueue = "payment.booking-events.dlq.v1"

var ErrInvoiceBroker = errors.New("payment invoice event broker unavailable")

type InvoiceHandler interface {
	Process(context.Context, []byte) error
}

type InvoiceConsumer struct {
	url        string
	handler    InvoiceHandler
	log        *slog.Logger
	retryDelay time.Duration
}

func NewInvoiceConsumer(url string, handler InvoiceHandler, log *slog.Logger) *InvoiceConsumer {
	return &InvoiceConsumer{url: url, handler: handler, log: log, retryDelay: time.Second}
}

func (c *InvoiceConsumer) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := c.consume(ctx); err != nil && ctx.Err() == nil {
			c.log.Warn("payment invoice event connection unavailable", "operation", "events.consume")
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

func declareInvoiceTopology(channel *amqp.Channel) error {
	if err := channel.ExchangeDeclare(Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(InvoiceQueue, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		return err
	}
	if err := channel.QueueBind(InvoiceQueue, model.ConfirmedEventType, Exchange, false, nil); err != nil {
		return err
	}
	if err := channel.ExchangeDeclare(InvoiceDeadExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := channel.QueueDeclare(InvoiceDeadQueue, true, false, false, false, amqp.Table{"x-queue-type": "classic"}); err != nil {
		return err
	}
	if err := channel.QueueBind(InvoiceDeadQueue, InvoiceQueue, InvoiceDeadExchange, false, nil); err != nil {
		return err
	}
	if err := channel.Qos(1, 0, false); err != nil {
		return err
	}
	return channel.Confirm(false)
}

func (c *InvoiceConsumer) consume(ctx context.Context) error {
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
		return ErrInvoiceBroker
	}
	defer func() {
		// Closing the socket also unblocks any pending AMQP RPC on shutdown/error.
		_ = socket.Close()
		if err := connection.CloseDeadline(time.Now().Add(time.Second)); err != nil && !errors.Is(err, amqp.ErrClosed) {
			c.log.Warn("payment invoice event close failed", "operation", "events.close")
		}
	}()
	channel, err := connection.Channel()
	if err != nil {
		return ErrInvoiceBroker
	}
	if err := declareInvoiceTopology(channel); err != nil {
		return ErrInvoiceBroker
	}
	returns := channel.NotifyReturn(make(chan amqp.Return, 1))
	confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	deliveries, err := channel.ConsumeWithContext(ctx, InvoiceQueue, "", false, false, false, false, nil)
	if err != nil {
		return ErrInvoiceBroker
	}
	if setup.Err() != nil {
		return ErrInvoiceBroker
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
				return ErrInvoiceBroker
			}
			err := c.handleDelivery(ctx, delivery, func(ctx context.Context, delivery amqp.Delivery, code string) error {
				return publishInvoiceDeadLetter(ctx, channel, socket, confirms, returns, delivery, code)
			})
			if err != nil {
				return err
			}
		}
	}
}

func (c *InvoiceConsumer) handleDelivery(ctx context.Context, delivery amqp.Delivery, deadLetter func(context.Context, amqp.Delivery, string) error) error {
	event, err := model.DecodeConfirmedEvent(delivery.Body)
	if err == nil && (delivery.ContentType != "application/json" || delivery.Type != model.ConfirmedEventType || delivery.RoutingKey != model.ConfirmedEventType || delivery.MessageId != event.EventID || delivery.CorrelationId != event.CorrelationID) {
		err = model.ErrInvalidConfirmation
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
			if errors.Is(err, model.ErrInvalidConfirmation) || errors.Is(err, repository.ErrInvoiceConflict) || errors.Is(err, repository.ErrInvoiceProfileMissing) || errors.Is(err, booking.ErrNotFound) || errors.Is(err, booking.ErrNotConfirmed) {
				code = "INVOICE_REJECTED"
				if errors.Is(err, repository.ErrInvoiceProfileMissing) {
					code = "CUSTOMER_SNAPSHOT_MISSING"
				}
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
	c.log.Warn("payment invoice event retained in DLQ", "operation", "events.dead_letter", "error_code", code)
	return delivery.Ack(false)
}

func publishInvoiceDeadLetter(ctx context.Context, channel *amqp.Channel, socket net.Conn, confirms <-chan amqp.Confirmation, returns <-chan amqp.Return, delivery amqp.Delivery, code string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = socket.Close() })
	defer stop()
	sequence := channel.GetNextPublishSeqNo()
	err := channel.PublishWithContext(ctx, InvoiceDeadExchange, InvoiceQueue, true, false, amqp.Publishing{ContentType: delivery.ContentType, DeliveryMode: amqp.Persistent,
		MessageId: delivery.MessageId, Type: delivery.Type, CorrelationId: delivery.CorrelationId, Body: delivery.Body, Headers: amqp.Table{"error_code": code}})
	if err != nil {
		return ErrInvoiceBroker
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-returns:
		return ErrInvoiceBroker
	case confirm, ok := <-confirms:
		if !ok || !confirm.Ack || confirm.DeliveryTag != sequence {
			return ErrInvoiceBroker
		}
		select {
		case <-returns:
			return ErrInvoiceBroker
		default:
		}
		return nil
	}
}
