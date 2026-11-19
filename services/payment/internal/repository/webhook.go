package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"payment/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *PaymentRepository) ReceiveWebhook(ctx context.Context, orderID, hash string) (model.PaymentAttempt, model.WebhookReceipt, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var attempt model.PaymentAttempt
	var receipt model.WebhookReceipt
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO payment.webhook_receipts (provider_order_id, payload_hash)
			VALUES (?, ?) ON CONFLICT (payload_hash) DO NOTHING`, orderID, hash).Error; err != nil {
			return err
		}
		if err := tx.Raw("SELECT * FROM payment.webhook_receipts WHERE payload_hash = ? FOR UPDATE", hash).Scan(&receipt).Error; err != nil {
			return err
		}
		if receipt.Status != "received" {
			return nil
		}
		err := tx.Table("payment.payment_attempts").Where("provider_order_id = ?", orderID).Take(&attempt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			receipt.Status = "unmatched"
			return tx.Exec(`UPDATE payment.webhook_receipts SET status = 'unmatched', processed_at = NOW() WHERE id = ?`, receipt.ID).Error
		}
		if err != nil {
			return err
		}
		receipt.PaymentID = &attempt.ID
		return tx.Exec("UPDATE payment.webhook_receipts SET payment_id = ? WHERE id = ?", attempt.ID, receipt.ID).Error
	})
	if err != nil {
		return attempt, receipt, false, ErrUnavailable
	}
	return attempt, receipt, receipt.Status != "received", nil
}

func (r *PaymentRepository) WebhookUnavailable(ctx context.Context, receiptID string, observation model.ProviderObservation) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt model.PaymentAttempt
		if err := tx.Raw("SELECT * FROM payment.payment_attempts WHERE id = ? FOR UPDATE", observation.PaymentID).Scan(&attempt).Error; err != nil {
			return err
		}
		var receipt model.WebhookReceipt
		if err := tx.Raw("SELECT * FROM payment.webhook_receipts WHERE id = ? FOR UPDATE", receiptID).Scan(&receipt).Error; err != nil {
			return err
		}
		if receipt.Status != "received" {
			return nil
		}
		if err := insertObservation(tx, observation); err != nil {
			return err
		}
		return tx.Exec("UPDATE payment.webhook_receipts SET error_code = ? WHERE id = ?", observation.ErrorCode, receiptID).Error
	})
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

// receiptID is empty when the later reconciliation worker applies a status lookup.
// External HTTP verification must finish before this transaction begins.
func (r *PaymentRepository) ApplyVerifiedObservation(ctx context.Context, paymentID, receiptID string, outcome model.PaymentOutcome) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt model.PaymentAttempt
		if err := tx.Raw("SELECT * FROM payment.payment_attempts WHERE id = ? FOR UPDATE", paymentID).Scan(&attempt).Error; err != nil {
			return err
		}
		if attempt.ID == "" {
			return gorm.ErrRecordNotFound
		}
		if receiptID != "" {
			var receipt model.WebhookReceipt
			if err := tx.Raw("SELECT * FROM payment.webhook_receipts WHERE id = ? FOR UPDATE", receiptID).Scan(&receipt).Error; err != nil {
				return err
			}
			if receipt.Status != "received" {
				return nil
			}
		}
		// Charge persistence may have attached a reference while GET status was running.
		if attempt.ProviderTransactionID != nil && outcome.TransactionID != "" && *attempt.ProviderTransactionID != outcome.TransactionID {
			reason := "PROVIDER_REFERENCE_MISMATCH"
			outcome.Status, outcome.ReviewReason = "", &reason
			outcome.Observation.VerificationResult, outcome.Observation.ErrorCode = "rejected", &reason
		}
		if err := insertObservation(tx, outcome.Observation); err != nil {
			return err
		}
		changes := map[string]any{"updated_at": outcome.VerifiedAt}
		if outcome.ReviewReason != nil {
			changes["review_reason"] = outcome.ReviewReason
		}
		if outcome.Status != "" {
			switch {
			case attempt.Status == "paid":
				if outcome.Status != "paid" {
					changes["review_reason"] = "PROVIDER_STATUS_CONTRADICTION"
				}
			case outcome.Status == "paid":
				changes["status"], changes["verified_at"] = "paid", outcome.VerifiedAt
				changes["provider_status"], changes["provider_transaction_id"] = outcome.ProviderStatus, outcome.TransactionID
				if attempt.Status == "expired" || attempt.Status == "failed" {
					changes["review_reason"] = "PAID_AFTER_TERMINAL_STATUS"
				}
				if err := insertPaidEvent(tx, attempt, outcome, receiptID); err != nil {
					return err
				}
			case attempt.Status == "initiating" || attempt.Status == "pending":
				changes["status"], changes["provider_status"] = outcome.Status, outcome.ProviderStatus
				changes["provider_transaction_id"] = outcome.TransactionID
			}
		}
		if err := tx.Table("payment.payment_attempts").Where("id = ?", attempt.ID).Updates(changes).Error; err != nil {
			return err
		}
		if receiptID == "" {
			return nil
		}
		status := "processed"
		if outcome.Observation.VerificationResult == "rejected" {
			status = "rejected"
		}
		return tx.Exec(`UPDATE payment.webhook_receipts SET status = ?, verified_at = ?, processed_at = ?, error_code = ?
			WHERE id = ?`, status, outcome.VerifiedAt, outcome.VerifiedAt, outcome.Observation.ErrorCode, receiptID).Error
	})
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

func insertObservation(tx *gorm.DB, observation model.ProviderObservation) error {
	return tx.Exec(`INSERT INTO payment.provider_observations
		(payment_id, source, provider_transaction_id, provider_status, amount_idr, currency,
		verification_result, payload_digest, error_code, request_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		observation.PaymentID, observation.Source, observation.ProviderTransactionID, observation.ProviderStatus,
		observation.AmountIDR, observation.Currency, observation.VerificationResult, observation.PayloadDigest,
		observation.ErrorCode, observation.RequestID).Error
}

func insertPaidEvent(tx *gorm.DB, attempt model.PaymentAttempt, outcome model.PaymentOutcome, receiptID string) error {
	id, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	const eventType = "payment.paid.v1"
	var cause any
	if receiptID != "" {
		cause = receiptID
	}
	envelope, err := json.Marshal(map[string]any{
		"event_id": id.String(), "event_type": eventType, "schema_version": 1,
		"aggregate_id": attempt.ID, "occurred_at": outcome.VerifiedAt,
		"correlation_id": outcome.Observation.RequestID, "causation_id": cause,
		"payload": map[string]any{"payment_id": attempt.ID, "booking_id": attempt.BookingID,
			"provider_order_id": attempt.ProviderOrderID, "amount_idr": attempt.AmountIDR, "currency": attempt.Currency,
			"verified_at": outcome.VerifiedAt, "provider_paid_at": nil},
	})
	if err != nil {
		return err
	}
	return tx.Exec(`INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at)
		VALUES (?, ?, ?, ?, ?)`, id.String(), attempt.ID, eventType, string(envelope), outcome.VerifiedAt).Error
}
