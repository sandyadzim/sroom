package checkout

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"booking/internal/payment"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) paidEventProcessed(ctx context.Context, event payment.PaidEvent) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var row struct{ PayloadHash string }
	err := r.db.WithContext(ctx).Table("booking.inbox_events").Where("consumer_name = ? AND event_id = ?", PaidConsumer, event.EventID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, ErrDatabaseUnavailable
	}
	if row.PayloadHash != event.Hash {
		return false, ErrConfirmationConflict
	}
	return true, nil
}

func (r *Repository) confirmVerifiedPayment(ctx context.Context, event payment.PaidEvent, confirmed payment.Confirmation, now func() time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var booking struct {
			PaymentContext
			ConfirmedPaymentID       *string
			ConfirmedProviderOrderID *string
		}
		err := tx.Table("booking.bookings").Select(paymentContextColumns+", confirmed_payment_id, confirmed_provider_order_id").
			Where("id = ?", event.Payload.BookingID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBookingNotFound
		}
		if err != nil {
			return err
		}
		at := now().UTC().Truncate(time.Microsecond)
		inserted := tx.Exec(`INSERT INTO booking.inbox_events (consumer_name,event_id,event_type,aggregate_id,booking_id,payload_hash,processed_at)
   VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT (consumer_name,event_id) DO NOTHING`, PaidConsumer, event.EventID, event.EventType, event.AggregateID, booking.BookingID, event.Hash, at)
		if inserted.Error != nil {
			return inserted.Error
		}
		if inserted.RowsAffected == 0 {
			var hash string
			if err := tx.Raw("SELECT payload_hash FROM booking.inbox_events WHERE consumer_name = ? AND event_id = ?", PaidConsumer, event.EventID).Scan(&hash).Error; err != nil {
				return err
			}
			if hash != event.Hash {
				return ErrConfirmationConflict
			}
			return nil
		}
		if booking.Status == "confirmed" {
			if booking.ConfirmedPaymentID != nil && *booking.ConfirmedPaymentID == confirmed.PaymentID &&
				booking.ConfirmedProviderOrderID != nil && *booking.ConfirmedProviderOrderID == confirmed.ProviderOrderID &&
				booking.TotalIDR == confirmed.AmountIDR && booking.Currency == confirmed.Currency && booking.CustomerID == confirmed.CustomerID {
				return nil
			}
			return ErrConfirmationConflict
		}
		if booking.Status == "payment_review" {
			return nil
		}
		var allocation struct {
			ID              string
			BlocksInventory bool
			MatchesPeriod   bool
		}
		if err := tx.Raw(`SELECT a.id,a.blocks_inventory,a.occupied_period = tstzrange(b.pickup_at,b.return_at,'[)') AS matches_period
   FROM booking.allocations a JOIN booking.bookings b ON b.id = a.booking_id WHERE a.booking_id = ? FOR UPDATE OF a`, booking.BookingID).Scan(&allocation).Error; err != nil {
			return err
		}
		reason := ""
		switch {
		case booking.Status != "pending_payment":
			reason = "BOOKING_NOT_CONFIRMABLE"
		case booking.CustomerID != confirmed.CustomerID || booking.TotalIDR != confirmed.AmountIDR || booking.Currency != confirmed.Currency || !booking.HoldExpiresAt.Equal(confirmed.HoldExpiresAt):
			reason = "PAYMENT_SNAPSHOT_MISMATCH"
		case confirmed.ReviewRequired:
			reason = "PAYMENT_REQUIRES_REVIEW"
		case !confirmed.VerifiedAt.Before(booking.HoldExpiresAt):
			reason = "PAYMENT_VERIFIED_TOO_LATE"
		case allocation.ID == "" || !allocation.BlocksInventory || !allocation.MatchesPeriod:
			reason = "ALLOCATION_UNAVAILABLE"
		}
		target := "confirmed"
		if reason != "" {
			target = "payment_review"
		} else {
			reason = "PAYMENT_CONFIRMED"
		}
		changes := map[string]any{"status": target, "updated_at": at}
		if target == "confirmed" {
			changes["confirmed_payment_id"], changes["confirmed_provider_order_id"] = confirmed.PaymentID, confirmed.ProviderOrderID
		}
		if err := tx.Table("booking.bookings").Where("id = ?", booking.BookingID).Updates(changes).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO booking.booking_status_history
   (booking_id,from_status,to_status,actor_type,reason_code,request_id,occurred_at)
   VALUES (?, ?, ?, 'provider_event', ?, ?, ?)`, booking.BookingID, booking.Status, target, reason, event.CorrelationID, at).Error; err != nil {
			return err
		}
		if target != "confirmed" {
			return nil
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		envelope, err := json.Marshal(map[string]any{"event_id": id.String(), "event_type": "booking.confirmed.v1", "schema_version": 1,
			"occurred_at": at, "aggregate_id": booking.BookingID, "correlation_id": event.CorrelationID, "causation_id": event.EventID,
			"payload": map[string]any{"booking_id": booking.BookingID, "payment_id": confirmed.PaymentID, "customer_id": booking.CustomerID,
				"confirmed_at": at, "total_idr": booking.TotalIDR, "currency": booking.Currency}})
		if err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO booking.outbox_events (id,aggregate_id,event_type,envelope,occurred_at)
   VALUES (?, ?, 'booking.confirmed.v1', CAST(? AS JSONB), ?)`, id.String(), booking.BookingID, string(envelope), at).Error
	})
	if errors.Is(err, ErrBookingNotFound) || errors.Is(err, ErrConfirmationConflict) {
		return err
	}
	if err != nil {
		return ErrDatabaseUnavailable
	}
	return nil
}
