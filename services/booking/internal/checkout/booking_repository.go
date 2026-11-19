package checkout

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"
)

const bookingColumns = `b.id, b.variant_id, b.vehicle_label_snapshot AS vehicle_label, b.status,
	b.pickup_at, b.return_at, b.rental_days, b.daily_rate_idr, b.total_idr, b.currency,
	b.hold_expires_at, b.terms_version, b.terms_accepted_at, b.created_at`

func (r *Repository) listBookings(ctx context.Context, customerID string, filters BookingFilters) ([]StaffBooking, BookingPage, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rows := []StaffBooking{}
	page := BookingPage{Number: filters.Page, Limit: filters.Limit}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := bookingFilterQuery(tx, customerID, filters)
		if err := query.Count(&page.Total).Error; err != nil {
			return err
		}
		return selectBookingRows(query, customerID).
			Order("b.created_at DESC, b.id DESC").
			Offset((filters.Page - 1) * filters.Limit).
			Limit(filters.Limit).
			Find(&rows).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, BookingPage{}, ErrDatabaseUnavailable
	}
	return rows, page, nil
}

func bookingFilterQuery(db *gorm.DB, customerID string, filters BookingFilters) *gorm.DB {
	query := db.Table("booking.bookings AS b")
	if customerID != "" {
		query = query.Where("b.customer_id = ?", customerID)
	}
	if filters.Status != "" {
		query = query.Where("b.status = ?", filters.Status)
	}
	if filters.VariantID != "" {
		query = query.Where("b.variant_id = ?", filters.VariantID)
	}
	if filters.PickupDate != "" {
		query = query.Where("b.pickup_local_date = ?", filters.PickupDate)
	}
	if filters.Search != "" {
		query = query.Where(`(strpos(lower(b.id::text), lower(?)) > 0 OR
			strpos(lower(b.customer_id::text), lower(?)) > 0 OR
			strpos(lower(b.vehicle_label_snapshot), lower(?)) > 0)`,
			filters.Search, filters.Search, filters.Search)
	}
	return query
}

func selectBookingRows(query *gorm.DB, customerID string) *gorm.DB {
	if customerID != "" {
		return query.Select(bookingColumns)
	}
	return query.Select(bookingColumns + ", b.customer_id, a.unit_id").
		Joins("LEFT JOIN booking.allocations AS a ON a.booking_id = b.id")
}

func (r *Repository) findBooking(ctx context.Context, customerID, bookingID string) (StaffBooking, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var booking StaffBooking
	query := bookingFilterQuery(r.db.WithContext(ctx), customerID, BookingFilters{})
	err := selectBookingRows(query, customerID).Where("b.id = ?", bookingID).Take(&booking).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return StaffBooking{}, ErrBookingNotFound
	case err != nil:
		return StaffBooking{}, ErrDatabaseUnavailable
	default:
		return booking, nil
	}
}

func (r *Repository) statusHistory(ctx context.Context, bookingID string) ([]StatusChange, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	history := []StatusChange{}
	err := r.db.WithContext(ctx).Table("booking.booking_status_history").
		Select("id", "from_status", "to_status", "actor_type", "reason_code", "occurred_at").
		Where("booking_id = ?", bookingID).
		Order("occurred_at ASC, id ASC").
		Find(&history).Error
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	return history, nil
}
