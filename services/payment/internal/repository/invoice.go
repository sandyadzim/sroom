package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"payment/internal/booking"
	"payment/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvoiceNotFound = errors.New("invoice not found")
var ErrInvoiceConflict = errors.New("invoice confirmation references conflict")
var ErrInvoiceProfileMissing = errors.New("customer billing snapshot is missing")

func (r *PaymentRepository) FindInvoice(ctx context.Context, bookingID string) (model.Invoice, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var invoice model.Invoice
	err := r.db.WithContext(ctx).Table("payment.invoices").Where("booking_id = ?", bookingID).Take(&invoice).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return invoice, ErrInvoiceNotFound
	}
	if err != nil {
		return invoice, ErrUnavailable
	}
	return invoice, nil
}

func (r *PaymentRepository) InvoiceEventProcessed(ctx context.Context, event model.ConfirmedEvent) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var row struct{ PayloadHash string }
	err := r.db.WithContext(ctx).Table("payment.inbox_events").Where("consumer_name = ? AND event_id = ?", model.InvoiceConsumer, event.EventID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	if row.PayloadHash != event.Hash {
		return false, ErrInvoiceConflict
	}
	return true, nil
}

func (r *PaymentRepository) IssueInvoice(ctx context.Context, event model.ConfirmedEvent, snapshot booking.InvoiceContext, now func() time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var attempt model.PaymentAttempt
		err := tx.Table("payment.payment_attempts").Where("id = ?", event.Payload.PaymentID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&attempt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvoiceConflict
		}
		if err != nil {
			return err
		}
		at := now().UTC().Truncate(time.Microsecond)
		insert := tx.Exec(`INSERT INTO payment.inbox_events (consumer_name,event_id,event_type,aggregate_id,payload_hash,processed_at)
   VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (consumer_name,event_id) DO NOTHING`, model.InvoiceConsumer, event.EventID, event.EventType, event.AggregateID, event.Hash, at)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			var hash string
			if err := tx.Raw("SELECT payload_hash FROM payment.inbox_events WHERE consumer_name = ? AND event_id = ?", model.InvoiceConsumer, event.EventID).Scan(&hash).Error; err != nil {
				return err
			}
			if hash != event.Hash {
				return ErrInvoiceConflict
			}
			return nil
		}
		if attempt.BookingID != snapshot.BookingID || attempt.CustomerID != snapshot.CustomerID || attempt.ProviderOrderID != snapshot.ProviderOrderID || attempt.AmountIDR != snapshot.TotalIDR || attempt.Currency != snapshot.Currency {
			return ErrInvoiceConflict
		}
		var existing model.Invoice
		err = tx.Table("payment.invoices").Where("booking_id = ?", snapshot.BookingID).Take(&existing).Error
		if err == nil {
			if existing.PaymentID != attempt.ID || existing.CustomerID != attempt.CustomerID || existing.TotalIDR != attempt.AmountIDR || existing.Currency != attempt.Currency {
				return ErrInvoiceConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if attempt.Status != "paid" || attempt.VerifiedAt == nil || attempt.VerifiedAt.IsZero() || attempt.ReviewReason != nil ||
			!attempt.VerifiedAt.Before(attempt.HoldExpiresAt) || attempt.VerifiedAt.After(snapshot.ConfirmedAt) {
			return ErrInvoiceConflict
		}
		if attempt.CustomerFullName == nil || attempt.CustomerEmail == nil || strings.TrimSpace(*attempt.CustomerEmail) == "" {
			return ErrInvoiceProfileMissing
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		invoice := model.Invoice{ID: id.String(), InvoiceNumber: "INV-" + strings.ToUpper(id.String()), BookingID: attempt.BookingID,
			PaymentID: attempt.ID, CustomerID: attempt.CustomerID, ProviderOrderID: attempt.ProviderOrderID,
			BilledToName: *attempt.CustomerFullName, BilledToEmail: *attempt.CustomerEmail,
			VariantID: snapshot.VariantID, VehicleLabel: snapshot.VehicleLabel, PickupAt: snapshot.PickupAt.UTC(), ReturnAt: snapshot.ReturnAt.UTC(),
			RentalDays: snapshot.RentalDays, DailyRateIDR: snapshot.DailyRateIDR, TotalIDR: attempt.AmountIDR, Currency: attempt.Currency, Bank: attempt.Bank,
			ConfirmedAt: snapshot.ConfirmedAt.UTC(), IssuedAt: at}
		return tx.Table("payment.invoices").Create(&invoice).Error
	})
	if errors.Is(err, ErrInvoiceConflict) || errors.Is(err, ErrInvoiceProfileMissing) {
		return err
	}
	if err != nil {
		return ErrUnavailable
	}
	return nil
}
