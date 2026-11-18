package checkout

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const holdRecordQuery = `
	SELECT request_hash, booking_id FROM booking.idempotency_records
	WHERE customer_id = ? AND idempotency_key = ?`

func (r *Repository) findHold(ctx context.Context, input HoldInput, hash string) (Hold, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	row := r.db.WithContext(ctx).Raw(holdRecordQuery, input.CustomerID, input.IdempotencyKey).Row()
	return savedHold(r.db.WithContext(ctx), row, hash)
}

func (r *Repository) rememberHoldKey(ctx context.Context, input HoldInput, hash string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO booking.idempotency_records (customer_id, idempotency_key, request_hash)
		VALUES (?, ?, ?) ON CONFLICT (customer_id, idempotency_key) DO NOTHING
	`, input.CustomerID, input.IdempotencyKey, hash).Error
	if err != nil {
		return ErrDatabaseUnavailable
	}
	return nil
}

func savedHold(db *gorm.DB, row *sql.Row, hash string) (Hold, error) {
	var storedHash string
	var bookingID sql.NullString
	err := row.Scan(&storedHash, &bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return Hold{}, nil
	}
	if err != nil {
		return Hold{}, ErrDatabaseUnavailable
	}
	if storedHash != hash {
		return Hold{}, ErrIdempotencyConflict
	}
	if !bookingID.Valid {
		return Hold{}, nil
	}

	var hold Hold
	err = db.Raw(`
		SELECT id, status, variant_id, vehicle_label_snapshot AS vehicle_label,
		       pickup_at, return_at, rental_days, daily_rate_idr, total_idr, currency, hold_expires_at
		FROM booking.bookings WHERE id = ?
	`, bookingID.String).Row().Scan(&hold.ID, &hold.Status, &hold.VariantID, &hold.VehicleLabel,
		&hold.PickupAt, &hold.ReturnAt, &hold.RentalDays, &hold.DailyRateIDR, &hold.TotalIDR,
		&hold.Currency, &hold.HoldExpiresAt)
	if err != nil {
		return Hold{}, ErrDatabaseUnavailable
	}
	return hold, nil
}

func (r *Repository) createHold(ctx context.Context, input HoldInput, hash string, hold Hold, unitID string, at time.Time, requestID string) (Hold, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := tx.Raw(holdRecordQuery+" FOR UPDATE", input.CustomerID, input.IdempotencyKey).Row()
		existing, err := savedHold(tx, row, hash)
		if err != nil {
			return err
		}
		if existing.ID != "" {
			hold = existing
			return nil
		}

		hold.ID = uuid.NewString()
		if err := tx.Exec(`
			INSERT INTO booking.bookings (
			    id, customer_id, variant_id, status, pickup_at, return_at, pickup_local_date, return_local_date,
			    vehicle_label_snapshot, rental_days, daily_rate_idr, total_idr, currency,
			    terms_version, terms_accepted_at, hold_expires_at, created_at, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, hold.ID, input.CustomerID, hold.VariantID, hold.Status, hold.PickupAt, hold.ReturnAt,
			hold.PickupAt.Format(time.DateOnly), hold.ReturnAt.Format(time.DateOnly), hold.VehicleLabel,
			hold.RentalDays, hold.DailyRateIDR, hold.TotalIDR, hold.Currency, input.TermsVersion,
			at, hold.HoldExpiresAt, at, at).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO booking.allocations (booking_id, unit_id, occupied_period)
			VALUES (?, ?, tstzrange(?, ?, '[)'))
		`, hold.ID, unitID, hold.PickupAt, hold.ReturnAt).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO booking.booking_status_history (booking_id, to_status, actor_type, occurred_at, request_id)
			VALUES (?, 'pending_payment', 'customer', ?, ?)
		`, hold.ID, at, requestID).Error; err != nil {
			return err
		}
		if err := insertBookingCreated(tx, hold, input.CustomerID, at, requestID); err != nil {
			return err
		}
		return tx.Exec(`
			UPDATE booking.idempotency_records SET booking_id = ?
			WHERE customer_id = ? AND idempotency_key = ?
		`, hold.ID, input.CustomerID, input.IdempotencyKey).Error
	})
	if err != nil {
		var postgresError *pgconn.PgError
		switch {
		case errors.As(err, &postgresError) && postgresError.Code == "23P01" &&
			postgresError.ConstraintName == "allocations_no_overlap":
			return Hold{}, ErrAvailabilityConflict
		case errors.Is(err, ErrIdempotencyConflict):
			return Hold{}, err
		default:
			return Hold{}, ErrDatabaseUnavailable
		}
	}
	return hold, nil
}

func insertBookingCreated(tx *gorm.DB, hold Hold, customerID string, at time.Time, requestID string) error {
	const eventType = "booking.created.v1"
	eventID := uuid.NewString()
	envelope, err := json.Marshal(map[string]any{
		"event_id": eventID, "event_type": eventType, "schema_version": 1, "occurred_at": at,
		"aggregate_id": hold.ID, "correlation_id": requestID, "causation_id": nil,
		"payload": map[string]any{
			"booking_id": hold.ID, "customer_id": customerID, "amount_idr": hold.TotalIDR,
			"currency": hold.Currency, "hold_expires_at": hold.HoldExpiresAt,
		},
	})
	if err != nil {
		return err
	}
	return tx.Exec(`
		INSERT INTO booking.outbox_events (id, aggregate_id, event_type, envelope, occurred_at)
		VALUES (?, ?, ?, CAST(? AS JSONB), ?)
	`, eventID, hold.ID, eventType, string(envelope), at).Error
}
