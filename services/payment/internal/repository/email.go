package repository

import (
	"context"
	"errors"
	"time"

	"payment/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func queueInvoiceEmails(tx *gorm.DB, invoice model.Invoice, at time.Time) error {
	jobs := make([]model.EmailDelivery, 0, 2)
	for _, kind := range []string{"booking_confirmation", "invoice"} {
		id, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		jobs = append(jobs, model.EmailDelivery{ID: id.String(), InvoiceID: invoice.ID, BookingID: invoice.BookingID,
			Kind: kind, Recipient: invoice.BilledToEmail, Status: "queued", NextAttemptAt: &at,
			MessageID: "<" + id.String() + "@sroom.test>", CreatedAt: at, UpdatedAt: at})
	}
	return tx.Table("payment.email_deliveries").Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "booking_id"}, {Name: "kind"}}, DoNothing: true,
	}).Create(&jobs).Error
}

func (r *PaymentRepository) RecoverEmailClaims(ctx context.Context, now time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// Expired owners cannot finish after recovery: results also check attempt_count.
	for _, terminal := range []bool{true, false} {
		query := r.db.WithContext(ctx).Table("payment.email_deliveries").Where("status = 'sending' AND locked_until <= ?", now)
		changes := map[string]any{"locked_until": nil, "last_error_code": "DELIVERY_INTERRUPTED", "updated_at": now}
		if terminal {
			query = query.Where("attempt_count >= ?", model.MaxEmailAttempts)
			changes["status"], changes["next_attempt_at"] = "failed", nil
		} else {
			query = query.Where("attempt_count < ?", model.MaxEmailAttempts)
			changes["status"], changes["next_attempt_at"] = "queued", now
		}
		if err := query.Updates(changes).Error; err != nil {
			return ErrUnavailable
		}
	}
	return nil
}

func (r *PaymentRepository) ClaimEmail(ctx context.Context, now time.Time) (model.EmailDelivery, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var job model.EmailDelivery
	claimed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Table("payment.email_deliveries").Where("status = 'queued' AND next_attempt_at <= ? AND attempt_count < ?", now, model.MaxEmailAttempts).
			Order("next_attempt_at ASC, id ASC").Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Take(&job).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		until := now.Add(30 * time.Second)
		job.AttemptCount++
		if err := tx.Table("payment.email_deliveries").Where("id = ?", job.ID).Updates(map[string]any{
			"status": "sending", "attempt_count": job.AttemptCount, "next_attempt_at": nil, "locked_until": until, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		job.Status, job.NextAttemptAt, job.LockedUntil = "sending", nil, &until
		claimed = true
		return nil
	})
	if err != nil {
		return model.EmailDelivery{}, false, ErrUnavailable
	}
	return job, claimed, nil
}

func (r *PaymentRepository) FinishEmail(ctx context.Context, job model.EmailDelivery, code string, now time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	changes := map[string]any{"locked_until": nil, "next_attempt_at": nil, "updated_at": now}
	if code == "" {
		changes["status"], changes["sent_at"], changes["last_error_code"] = "sent", now, nil
	} else {
		changes["last_error_code"] = code
		if code == "INVALID_RECIPIENT" || code == "INVALID_MESSAGE" || job.AttemptCount >= model.MaxEmailAttempts {
			changes["status"] = "failed"
		} else {
			delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
			changes["status"], changes["next_attempt_at"] = "queued", now.Add(delays[job.AttemptCount-1])
		}
	}
	result := r.db.WithContext(ctx).Table("payment.email_deliveries").Where("id = ? AND status = 'sending' AND attempt_count = ?", job.ID, job.AttemptCount).Updates(changes)
	if result.Error != nil {
		return false, ErrUnavailable
	}
	return result.RowsAffected == 1, nil
}

func (r *PaymentRepository) EmailDeliveries(ctx context.Context, bookingID string) ([]model.EmailDelivery, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	jobs := make([]model.EmailDelivery, 0, 2)
	err := r.db.WithContext(ctx).Table("payment.email_deliveries").Select("kind,status,attempt_count,next_attempt_at,sent_at,last_error_code").Where("booking_id = ?", bookingID).Order("kind ASC").Find(&jobs).Error
	if err != nil {
		return nil, ErrUnavailable
	}
	return jobs, nil
}
