package checkout

import (
	"context"
	"time"

	"gorm.io/gorm"
)

func (r *Repository) PendingOutbox(ctx context.Context) ([]OutboxEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	events := make([]OutboxEvent, 0)
	err := r.db.WithContext(ctx).Table("booking.outbox_events").
		Where("published_at IS NULL AND event_type = ? AND (last_error_code IS NULL OR last_error_code <> ?)", "booking.confirmed.v1", "INVALID_EVENT").
		Order("occurred_at ASC, id ASC").Limit(20).Find(&events).Error
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	return events, nil
}

func (r *Repository) StartOutboxPublish(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result := r.db.WithContext(ctx).Table("booking.outbox_events").Where("id = ? AND published_at IS NULL", id).
		Update("publish_attempts", gorm.Expr("publish_attempts + 1"))
	if result.Error != nil || result.RowsAffected != 1 {
		return ErrDatabaseUnavailable
	}
	return nil
}

func (r *Repository) MarkOutboxFailed(ctx context.Context, id, code string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result := r.db.WithContext(ctx).Table("booking.outbox_events").Where("id = ? AND published_at IS NULL", id).Update("last_error_code", code)
	if result.Error != nil || result.RowsAffected != 1 {
		return ErrDatabaseUnavailable
	}
	return nil
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result := r.db.WithContext(ctx).Table("booking.outbox_events").Where("id = ? AND published_at IS NULL", id).
		Updates(map[string]any{"published_at": gorm.Expr("NOW()"), "last_error_code": nil})
	if result.Error != nil || result.RowsAffected != 1 {
		return ErrDatabaseUnavailable
	}
	return nil
}
