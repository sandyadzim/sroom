package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrInitiationClosed = errors.New("payment initiation is closed")

type InitiationState struct {
	ClosedAt  *time.Time
	PaymentID *string
}

// Reserve and expiry both acquire this lock before checking or creating an attempt.
func lockInitiationGate(tx *gorm.DB, bookingID string) (*time.Time, error) {
	if err := tx.Exec("INSERT INTO payment.initiation_gates (booking_id) VALUES (?) ON CONFLICT DO NOTHING", bookingID).Error; err != nil {
		return nil, err
	}
	var gate struct{ ClosedAt *time.Time }
	err := tx.Raw("SELECT closed_at FROM payment.initiation_gates WHERE booking_id = ? FOR UPDATE", bookingID).Scan(&gate).Error
	return gate.ClosedAt, err
}

// A closed gate without an attempt is a permanent promise that no charge can start.
func (r *PaymentRepository) CloseUnusedInitiation(ctx context.Context, bookingID string, at time.Time) (InitiationState, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var result InitiationState
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		closedAt, err := lockInitiationGate(tx, bookingID)
		if err != nil {
			return err
		}
		var attempt struct{ ID string }
		if err := tx.Raw("SELECT id FROM payment.payment_attempts WHERE booking_id = ?", bookingID).Scan(&attempt).Error; err != nil {
			return err
		}
		if attempt.ID != "" {
			result.PaymentID = &attempt.ID
			return nil
		}
		if closedAt == nil {
			at = at.UTC().Truncate(time.Microsecond)
			if err := tx.Exec("UPDATE payment.initiation_gates SET closed_at = ? WHERE booking_id = ?", at, bookingID).Error; err != nil {
				return err
			}
			closedAt = &at
		}
		result.ClosedAt = closedAt
		return nil
	})
	if err != nil {
		return InitiationState{}, ErrUnavailable
	}
	return result, nil
}
