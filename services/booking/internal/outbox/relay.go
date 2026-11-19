package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"booking/internal/checkout"
	"booking/internal/messaging"
)

type EventPublisher interface {
	Connect(context.Context) error
	Publish(context.Context, checkout.OutboxEvent) error
	Close() error
}

type OutboxRelay struct {
	repo      *checkout.Repository
	publisher EventPublisher
	log       *slog.Logger
}

func NewOutboxRelay(repo *checkout.Repository, publisher EventPublisher, log *slog.Logger) *OutboxRelay {
	return &OutboxRelay{repo: repo, publisher: publisher, log: log}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	defer func() {
		if err := r.publisher.Close(); err != nil {
			r.log.Warn("outbox connection close failed", "operation", "outbox.close")
		}
	}()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := r.ProcessBatch(ctx); err != nil && ctx.Err() == nil {
			r.log.Warn("outbox relay unavailable", "operation", "outbox.relay", "error_code", publicationErrorCode(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelay) ProcessBatch(ctx context.Context) error {
	if err := r.publisher.Connect(ctx); err != nil {
		return err
	}
	events, err := r.repo.PendingOutbox(ctx)
	if err != nil {
		return err
	}
	for _, event := range events {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !validateOutboxEvent(&event) {
			if err := r.repo.MarkOutboxFailed(ctx, event.ID, "INVALID_EVENT"); err != nil {
				return err
			}
			r.log.Warn("invalid outbox event retained", "operation", "outbox.validate", "event_id", event.ID, "error_code", "INVALID_EVENT")
			continue
		}
		if err := r.repo.StartOutboxPublish(ctx, event.ID); err != nil {
			return err
		}
		publishContext, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := r.publisher.Publish(publishContext, event)
		cancel()
		if err != nil {
			recordError := r.repo.MarkOutboxFailed(ctx, event.ID, publicationErrorCode(err))
			if closeError := r.publisher.Close(); closeError != nil {
				r.log.Warn("outbox connection close failed", "operation", "outbox.close")
			}
			if recordError != nil {
				return recordError
			}
			return err
		}
		if err := r.repo.MarkOutboxPublished(ctx, event.ID); err != nil {
			return err
		}
		r.log.Info("outbox event published", "operation", "outbox.publish", "event_id", event.ID, "event_type", event.EventType, "request_id", event.CorrelationID)
	}
	return nil
}

func publicationErrorCode(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "PUBLISH_TIMEOUT"
	case errors.Is(err, messaging.ErrUnroutable):
		return "UNROUTABLE"
	case errors.Is(err, messaging.ErrTopology):
		return "BROKER_TOPOLOGY_UNAVAILABLE"
	case errors.Is(err, messaging.ErrBrokerUnavailable):
		return "BROKER_UNAVAILABLE"
	case errors.Is(err, messaging.ErrPublishNack):
		return "PUBLISH_REJECTED"
	case errors.Is(err, checkout.ErrDatabaseUnavailable):
		return "DATABASE_UNAVAILABLE"
	default:
		return "PUBLISH_FAILED"
	}
}
