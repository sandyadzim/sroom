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

func (r *Repository) expireDemoHold(ctx context.Context, bookingID string, now func() time.Time, requestID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	expired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var booking PaymentContext
		err := tx.Table("booking.bookings").Select(paymentContextColumns).
			Where("id = ?", bookingID).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if booking.Status != "pending_payment" {
			return nil
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
			return ErrInvalidDemoAllocation
		}
		if err != nil {
			return err
		}
		// Recheck the deadline after both locks, just like demo confirmation.
		at := now().UTC()
		if at.Before(booking.HoldExpiresAt) {
			return nil
		}
		at = at.Truncate(time.Microsecond)
		if err := tx.Exec(`UPDATE booking.bookings SET status = 'expired', updated_at = ? WHERE id = ?`, at, bookingID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE booking.allocations SET blocks_inventory = false, released_at = ? WHERE id = ?`, at, allocationID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO booking.booking_status_history
			    (booking_id, from_status, to_status, actor_type, reason_code, request_id, occurred_at)
			VALUES (?, 'pending_payment', 'expired', 'system', 'DEMO_HOLD_EXPIRED', ?, ?)
		`, bookingID, requestID, at).Error; err != nil {
			return err
		}
		const eventType = "booking.demo_expired.v1"
		eventID := uuid.NewString()
		envelope, err := json.Marshal(map[string]any{
			"event_id": eventID, "event_type": eventType, "schema_version": 1,
			"occurred_at": at, "aggregate_id": booking.BookingID, "correlation_id": requestID, "causation_id": nil,
			"payload": map[string]any{
				"booking_id": booking.BookingID, "expired_at": at, "reason_code": "DEMO_HOLD_EXPIRED", "simulated": true,
			},
		})
		if err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO booking.outbox_events (id, aggregate_id, event_type, envelope, occurred_at)
			VALUES (?, ?, ?, CAST(? AS JSONB), ?)`, eventID, bookingID, eventType, string(envelope), at).Error; err != nil {
			return err
		}
		expired = true
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrInvalidDemoAllocation) {
			return false, err
		}
		return false, ErrDatabaseUnavailable
	}
	return expired, nil
}
