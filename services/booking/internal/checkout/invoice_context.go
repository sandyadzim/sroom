package checkout

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrInvoiceNotReady = errors.New("booking has no verified confirmation")

type InvoiceContext struct {
	BookingID           string    `json:"booking_id"`
	CustomerID          string    `json:"customer_id"`
	VariantID           string    `json:"variant_id"`
	VehicleLabel        string    `json:"vehicle_label"`
	Status              string    `json:"status"`
	PaymentID           string    `json:"payment_id"`
	ProviderOrderID     string    `json:"provider_order_id"`
	ConfirmationEventID string    `json:"confirmation_event_id"`
	ConfirmedAt         time.Time `json:"confirmed_at"`
	PickupAt            time.Time `json:"pickup_at"`
	ReturnAt            time.Time `json:"return_at"`
	RentalDays          int       `json:"rental_days"`
	DailyRateIDR        int64     `json:"daily_rate_idr" gorm:"column:daily_rate_idr"`
	TotalIDR            int64     `json:"total_idr" gorm:"column:total_idr"`
	Currency            string    `json:"currency"`
}

func (s *Service) InvoiceContext(ctx context.Context, bookingID string) (InvoiceContext, error) {
	result, err := s.allocations.invoiceContext(ctx, bookingID)
	if err != nil {
		return InvoiceContext{}, err
	}
	result.PickupAt, result.ReturnAt = result.PickupAt.In(s.wita), result.ReturnAt.In(s.wita)
	result.ConfirmedAt = result.ConfirmedAt.UTC()
	return result, nil
}

func (r *Repository) invoiceContext(ctx context.Context, bookingID string) (InvoiceContext, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var result InvoiceContext
	err := r.db.WithContext(ctx).Table("booking.bookings b").Select(`b.id AS booking_id,b.customer_id,b.variant_id,
 b.vehicle_label_snapshot AS vehicle_label,b.status,b.confirmed_payment_id AS payment_id,
 b.confirmed_provider_order_id AS provider_order_id,o.id AS confirmation_event_id,o.occurred_at AS confirmed_at,
 b.pickup_at,b.return_at,b.rental_days,b.daily_rate_idr,b.total_idr,b.currency`).
		Joins("LEFT JOIN booking.outbox_events o ON o.aggregate_id = b.id AND o.event_type = 'booking.confirmed.v1'").Where("b.id = ?", bookingID).Take(&result).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return InvoiceContext{}, ErrBookingNotFound
	}
	if err != nil {
		return InvoiceContext{}, ErrDatabaseUnavailable
	}
	if result.Status != "confirmed" || result.PaymentID == "" || result.ProviderOrderID == "" || result.ConfirmationEventID == "" || result.ConfirmedAt.IsZero() {
		return InvoiceContext{}, ErrInvoiceNotReady
	}
	return result, nil
}
