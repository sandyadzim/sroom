package checkout

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const demoConfirmedEvent = "booking.demo_confirmed.v1"

func (r *Repository) confirmDemoPayment(ctx context.Context, bookingID string, amountIDR int64, currency string, now func() time.Time, requestID string) (DemoConfirmation, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	result := DemoConfirmation{BookingID: bookingID, Status: "confirmed", Simulated: true}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var booking PaymentContext
		err := tx.Table("booking.bookings").Select(paymentContextColumns).
			Where("id = ?", bookingID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBookingNotFound
		}
		if err != nil {
			return err
		}
		result.BookingID = booking.BookingID
		if amountIDR != booking.TotalIDR || currency != booking.Currency {
			return ErrPaymentAmountMismatch
		}
		if booking.Status == "confirmed" {
			// The saved demo event distinguishes a retry from another confirmation source.
			err := tx.Raw(`SELECT occurred_at FROM booking.outbox_events
				WHERE aggregate_id = ? AND event_type = ?`, bookingID, demoConfirmedEvent).
				Row().Scan(&result.ConfirmedAt)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrBookingNotConfirmable
			}
			return err
		}
		if booking.Status != "pending_payment" {
			return ErrBookingNotConfirmable
		}
		var allocationID string
		err = tx.Raw(`
			SELECT a.id FROM booking.allocations a
			JOIN booking.bookings b ON b.id = a.booking_id
			WHERE a.booking_id = ? AND a.blocks_inventory
			  AND a.occupied_period = tstzrange(b.pickup_at, b.return_at, '[)')
			FOR UPDATE OF a
		`, bookingID).Row().Scan(&allocationID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBookingNotConfirmable
		}
		if err != nil {
			return err
		}
		// Check after acquiring locks so waiting cannot extend the hold window.
		at := now().UTC()
		if !at.Before(booking.HoldExpiresAt) {
			return ErrHoldExpired
		}
		// PostgreSQL stores microseconds. Use the same precision for the first response and retries.
		at = at.Truncate(time.Microsecond)
		result.ConfirmedAt = at
		if err := tx.Exec(`UPDATE booking.bookings SET status = 'confirmed', updated_at = ? WHERE id = ?`, at, bookingID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO booking.booking_status_history
			    (booking_id, from_status, to_status, actor_type, reason_code, request_id, occurred_at)
			VALUES (?, 'pending_payment', 'confirmed', 'system', 'DEMO_PAYMENT_CONFIRMED', ?, ?)
		`, bookingID, requestID, at).Error; err != nil {
			return err
		}
		eventID := uuid.NewString()
		envelope, err := json.Marshal(map[string]any{
			"event_id": eventID, "event_type": demoConfirmedEvent, "schema_version": 1,
			"occurred_at": at, "aggregate_id": booking.BookingID, "correlation_id": requestID, "causation_id": nil,
			"payload": map[string]any{
				"booking_id": booking.BookingID, "customer_id": booking.CustomerID, "confirmed_at": at,
				"total_idr": booking.TotalIDR, "currency": booking.Currency, "simulated": true,
			},
		})
		if err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO booking.outbox_events (id, aggregate_id, event_type, envelope, occurred_at)
			VALUES (?, ?, ?, CAST(? AS JSONB), ?)`, eventID, bookingID, demoConfirmedEvent, string(envelope), at).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrBookingNotFound), errors.Is(err, ErrPaymentAmountMismatch),
			errors.Is(err, ErrHoldExpired), errors.Is(err, ErrBookingNotConfirmable):
			return DemoConfirmation{}, err
		default:
			return DemoConfirmation{}, ErrDatabaseUnavailable
		}
	}
	result.ConfirmedAt = result.ConfirmedAt.UTC()
	return result, nil
}
