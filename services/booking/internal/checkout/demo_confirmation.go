package checkout

import (
	"context"
	"errors"
	"time"
)

var (
	ErrPaymentAmountMismatch = errors.New("payment amount or currency does not match the booking")
	ErrHoldExpired           = errors.New("booking hold deadline has been reached")
	ErrBookingNotConfirmable = errors.New("booking is not eligible for confirmation")
)

type DemoConfirmation struct {
	BookingID   string    `json:"booking_id"`
	Status      string    `json:"status"`
	ConfirmedAt time.Time `json:"confirmed_at"`
	Simulated   bool      `json:"simulated"`
}

func (s *Service) ConfirmDemoPayment(ctx context.Context, bookingID string, amountIDR int64, currency string, now func() time.Time, requestID string) (DemoConfirmation, error) {
	return s.allocations.confirmDemoPayment(ctx, bookingID, amountIDR, currency, now, requestID)
}
