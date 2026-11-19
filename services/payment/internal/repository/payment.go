package repository

import (
	"context"
	"errors"
	"time"

	"payment/internal/model"

	"gorm.io/gorm"
)

var (
	ErrUnavailable         = errors.New("payment database is unavailable")
	ErrIdempotencyConflict = errors.New("this idempotency key was used with a different request")
	ErrSnapshotMismatch    = errors.New("booking does not match the saved payment")
)

type PaymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) *PaymentRepository { return &PaymentRepository{db: db} }

func (r *PaymentRepository) Reserve(ctx context.Context, attempt model.PaymentAttempt, key, hash string) (model.PaymentAttempt, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var saved model.PaymentAttempt
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		closedAt, err := lockInitiationGate(tx, attempt.BookingID)
		if err != nil {
			return err
		}
		if closedAt != nil {
			return ErrInitiationClosed
		}
		insert := tx.Exec(`INSERT INTO payment.payment_attempts
			(id, booking_id, customer_id, provider_order_id, amount_idr, currency, hold_expires_at, customer_full_name, customer_email)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT (booking_id) DO NOTHING`,
			attempt.ID, attempt.BookingID, attempt.CustomerID, attempt.ProviderOrderID,
			attempt.AmountIDR, attempt.Currency, attempt.HoldExpiresAt, attempt.CustomerFullName, attempt.CustomerEmail)
		if insert.Error != nil {
			return insert.Error
		}
		created = insert.RowsAffected == 1
		if err := tx.Raw("SELECT * FROM payment.payment_attempts WHERE booking_id = ? FOR UPDATE", attempt.BookingID).Scan(&saved).Error; err != nil {
			return err
		}
		if saved.ID == "" || saved.CustomerID != attempt.CustomerID || saved.AmountIDR != attempt.AmountIDR ||
			saved.Currency != attempt.Currency || !saved.HoldExpiresAt.Equal(attempt.HoldExpiresAt) {
			return ErrSnapshotMismatch
		}
		if err := tx.Exec(`INSERT INTO payment.idempotency_records
			(customer_id, idempotency_key, request_hash, payment_id) VALUES (?, ?, ?, ?)
			ON CONFLICT (customer_id, idempotency_key) DO NOTHING`, attempt.CustomerID, key, hash, saved.ID).Error; err != nil {
			return err
		}
		var record struct{ RequestHash, PaymentID string }
		if err := tx.Raw(`SELECT request_hash, payment_id FROM payment.idempotency_records
			WHERE customer_id = ? AND idempotency_key = ? FOR UPDATE`, attempt.CustomerID, key).Scan(&record).Error; err != nil {
			return err
		}
		if record.RequestHash != hash || record.PaymentID != saved.ID {
			return ErrIdempotencyConflict
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrIdempotencyConflict) || errors.Is(err, ErrSnapshotMismatch) || errors.Is(err, ErrInitiationClosed) {
			return model.PaymentAttempt{}, false, err
		}
		return model.PaymentAttempt{}, false, ErrUnavailable
	}
	return saved, created, nil
}

func (r *PaymentRepository) SaveCharge(ctx context.Context, attempt model.PaymentAttempt, observation model.ProviderObservation) (model.PaymentAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var saved model.PaymentAttempt
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		changes := map[string]any{"updated_at": time.Now().UTC()}
		if attempt.ProviderExpiresAt != nil {
			changes["provider_expires_at"] = attempt.ProviderExpiresAt
		}
		if attempt.Status == "pending" {
			changes["status"] = "pending"
			changes["provider_transaction_id"] = attempt.ProviderTransactionID
			changes["provider_status"] = attempt.ProviderStatus
			changes["va_number"] = attempt.VANumber
		} else if attempt.ReviewReason != nil {
			changes["review_reason"] = attempt.ReviewReason
		}
		if err := tx.Table("payment.payment_attempts").Where("id = ? AND status = 'initiating' AND review_reason IS NULL", attempt.ID).Updates(changes).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO payment.provider_observations
			(payment_id, source, provider_transaction_id, provider_status, amount_idr, currency,
			 verification_result, payload_digest, error_code, request_id)
			VALUES (?, 'charge', ?, ?, ?, ?, ?, ?, ?, ?)`, observation.PaymentID,
			observation.ProviderTransactionID, observation.ProviderStatus, observation.AmountIDR, observation.Currency,
			observation.VerificationResult, observation.PayloadDigest, observation.ErrorCode, observation.RequestID).Error; err != nil {
			return err
		}
		return tx.Table("payment.payment_attempts").Where("id = ?", attempt.ID).Take(&saved).Error
	})
	if err != nil {
		return model.PaymentAttempt{}, ErrUnavailable
	}
	return saved, nil
}
