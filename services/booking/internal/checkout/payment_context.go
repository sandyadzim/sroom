package checkout

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type PaymentContext struct {
	BookingID     string    `json:"booking_id"`
	CustomerID    string    `json:"customer_id"`
	TotalIDR      int64     `json:"total_idr" gorm:"column:total_idr"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	HoldExpiresAt time.Time `json:"hold_expires_at"`
}

const paymentContextColumns = "id AS booking_id, customer_id, total_idr, currency, status, hold_expires_at"

func (s *Service) PaymentContext(ctx context.Context, bookingID string) (PaymentContext, error) {
	result, err := s.allocations.paymentContext(ctx, bookingID)
	result.HoldExpiresAt = result.HoldExpiresAt.UTC()
	return result, err
}

func (r *Repository) paymentContext(ctx context.Context, bookingID string) (PaymentContext, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var result PaymentContext
	err := r.db.WithContext(ctx).Table("booking.bookings").
		Select(paymentContextColumns).Where("id = ?", bookingID).Take(&result).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return PaymentContext{}, ErrBookingNotFound
	case err != nil:
		return PaymentContext{}, ErrDatabaseUnavailable
	default:
		return result, nil
	}
}

// DueHolds reads candidates for expiry evaluation; it does not release inventory.
func (r *Repository) DueHolds(ctx context.Context, checkedAt time.Time) ([]PaymentContext, error) {
	return r.dueHoldsAfter(ctx, checkedAt, HoldCursor{})
}

type HoldCursor struct {
	Deadline time.Time
	ID       string
}

func (r *Repository) dueHoldsAfter(ctx context.Context, checkedAt time.Time, cursor HoldCursor) ([]PaymentContext, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	results := []PaymentContext{}
	query := r.db.WithContext(ctx).Table("booking.bookings").
		Select(paymentContextColumns).
		Where("status = ? AND hold_expires_at <= ?", "pending_payment", checkedAt)
	if cursor.ID != "" {
		query = query.Where("(hold_expires_at, id) > (?, CAST(? AS UUID))", cursor.Deadline, cursor.ID)
	}
	err := query.Order("hold_expires_at ASC, id ASC").Limit(50).Find(&results).Error
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	for index := range results {
		results[index].HoldExpiresAt = results[index].HoldExpiresAt.UTC()
	}
	return results, nil
}
