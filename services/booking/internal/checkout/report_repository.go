package checkout

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// Demo confirmations have no canonical Payment reference and are excluded from value/conversion.
const verifiedConfirmation = "status = 'confirmed' AND confirmed_payment_id IS NOT NULL AND confirmed_provider_order_id IS NOT NULL"

func (r *Repository) readReport(ctx context.Context, period ReportPeriod, overview bool) (Report, []ReportVariant, []StaffBooking, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	report := Report{Period: period, Daily: []ReportDay{}}
	variants := []ReportVariant{}
	recent := []StaffBooking{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Raw(`SELECT count(*) AS total_bookings,
   count(*) FILTER (WHERE status = 'pending_payment') AS pending_payment_bookings,
   count(*) FILTER (WHERE status = 'confirmed') AS confirmed_bookings,
   count(*) FILTER (WHERE status = 'expired') AS expired_bookings,
   count(*) FILTER (WHERE status = 'payment_review') AS payment_review_bookings,
   count(*) FILTER (WHERE `+verifiedConfirmation+`) AS verified_confirmed_bookings,
   COALESCE(sum(total_idr) FILTER (WHERE `+verifiedConfirmation+`), 0) AS confirmed_value_idr
   FROM booking.bookings WHERE created_at >= ? AND created_at < ?`, period.Start, period.End).Scan(&report.Summary).Error
		if err != nil {
			return err
		}
		err = tx.Raw(`SELECT to_char(created_at AT TIME ZONE 'Asia/Makassar', 'YYYY-MM-DD') AS date,
   count(*) AS bookings,
   COALESCE(sum(total_idr) FILTER (WHERE `+verifiedConfirmation+`), 0) AS confirmed_value_idr
   FROM booking.bookings WHERE created_at >= ? AND created_at < ?
   GROUP BY date ORDER BY date`, period.Start, period.End).Scan(&report.Daily).Error
		if err != nil {
			return err
		}
		if overview {
			return selectBookingRows(tx.Table("booking.bookings AS b"), "").
				Where("b.created_at >= ? AND b.created_at < ?", period.Start, period.End).
				Order("b.created_at DESC, b.id DESC").Limit(5).Find(&recent).Error
		}
		return tx.Raw(`SELECT variant_id,
   (array_agg(vehicle_label_snapshot ORDER BY created_at DESC, id DESC))[1] AS vehicle_label,
   count(*) AS bookings FROM booking.bookings
   WHERE created_at >= ? AND created_at < ?
   GROUP BY variant_id ORDER BY bookings DESC, variant_id ASC LIMIT 10`, period.Start, period.End).Scan(&variants).Error
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Report{}, nil, nil, ErrDatabaseUnavailable
	}
	return report, variants, recent, nil
}
