package checkout

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"booking/internal/payment"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidExpiryAssessment = errors.New("expiry assessment does not authorize release")

func (r *Repository) expireUnusedHold(ctx context.Context, assessment payment.ExpiryAssessment, now func() time.Time, requestID string) (bool, error) {
	proofAt := assessment.ClosedAt
	reason := "NO_PAYMENT_ATTEMPT_CLOSED"
	switch assessment.Result {
	case "no_attempt_closed":
		if assessment.PaymentID != nil || assessment.ReleaseAuthorizedAt != nil {
			return false, ErrInvalidExpiryAssessment
		}
	case "provider_expired":
		if assessment.PaymentID == nil || len(*assessment.PaymentID) != 36 || uuid.Validate(*assessment.PaymentID) != nil ||
			*assessment.PaymentID == uuid.Nil.String() || assessment.ClosedAt != nil {
			return false, ErrInvalidExpiryAssessment
		}
		proofAt, reason = assessment.ReleaseAuthorizedAt, "PROVIDER_EXPIRED"
	default:
		return false, ErrInvalidExpiryAssessment
	}
	if proofAt == nil || proofAt.IsZero() || proofAt.Before(assessment.HoldExpiresAt) {
		return false, ErrInvalidExpiryAssessment
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	expired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var booking PaymentContext
		err := tx.Table("booking.bookings").Select(paymentContextColumns).Where("id = ?", assessment.BookingID).
			Clauses(clause.Locking{Strength: "UPDATE"}).Take(&booking).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if booking.Status != "pending_payment" {
			return nil
		}
		if !booking.HoldExpiresAt.Equal(assessment.HoldExpiresAt) {
			return ErrInvalidExpiryAssessment
		}
		var allocationID string
		err = tx.Raw(`SELECT a.id FROM booking.allocations a
   JOIN booking.bookings b ON b.id = a.booking_id
   WHERE a.booking_id = ? AND a.blocks_inventory
     AND a.occupied_period = tstzrange(b.pickup_at,b.return_at,'[)') FOR UPDATE OF a`, booking.BookingID).Row().Scan(&allocationID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidExpiryAssessment
		}
		if err != nil {
			return err
		}
		at := now().UTC().Truncate(time.Microsecond)
		if at.Before(booking.HoldExpiresAt) || at.Before(*proofAt) {
			return nil
		}
		if err := tx.Exec("UPDATE booking.bookings SET status = 'expired', updated_at = ? WHERE id = ?", at, booking.BookingID).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE booking.allocations SET blocks_inventory = false, released_at = ? WHERE id = ?", at, allocationID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO booking.booking_status_history
   (booking_id,from_status,to_status,actor_type,reason_code,request_id,occurred_at)
   VALUES (?, 'pending_payment','expired','system', ?, ?, ?)`, booking.BookingID, reason, requestID, at).Error; err != nil {
			return err
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		const eventType = "booking.expired.v1"
		payload := map[string]any{"booking_id": booking.BookingID, "expired_at": at, "reason_code": reason,
			"hold_expires_at": booking.HoldExpiresAt}
		if assessment.Result == "provider_expired" {
			payload["payment_id"], payload["release_authorized_at"] = assessment.PaymentID, assessment.ReleaseAuthorizedAt
		} else {
			payload["initiation_closed_at"] = assessment.ClosedAt
		}
		envelope, err := json.Marshal(map[string]any{"event_id": id.String(), "event_type": eventType, "schema_version": 1,
			"occurred_at": at, "aggregate_id": booking.BookingID, "correlation_id": requestID, "causation_id": nil,
			"payload": payload})
		if err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO booking.outbox_events (id,aggregate_id,event_type,envelope,occurred_at)
   VALUES (?, ?, ?, CAST(? AS JSONB), ?)`, id.String(), booking.BookingID, eventType, string(envelope), at).Error; err != nil {
			return err
		}
		expired = true
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrInvalidExpiryAssessment) {
			return false, err
		}
		return false, ErrDatabaseUnavailable
	}
	return expired, nil
}
