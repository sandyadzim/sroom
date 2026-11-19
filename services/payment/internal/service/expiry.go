package service

import (
	"context"
	"errors"

	"payment/internal/dto"
)

var ErrNotExpirable = errors.New("booking is not pending or its deadline has not been reached")

func (s *PaymentService) AssessExpiry(ctx context.Context, bookingID, requestID string) (dto.ExpiryAssessment, error) {
	// HTTP is outside the transaction. Booking rechecks its state before releasing.
	snapshot, err := s.readBooking(ctx, bookingID, requestID)
	if err != nil {
		return dto.ExpiryAssessment{}, err
	}
	at := s.now()
	if snapshot.Status != "pending_payment" || at.Before(snapshot.HoldExpiresAt) {
		return dto.ExpiryAssessment{}, ErrNotExpirable
	}
	state, err := s.repo.CloseUnusedInitiation(ctx, bookingID, at)
	if err != nil {
		return dto.ExpiryAssessment{}, err
	}
	result := "attempt_exists"
	if state.ClosedAt != nil {
		result = "no_attempt_closed"
	}
	return dto.ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: snapshot.HoldExpiresAt.UTC(),
		Result: result, ClosedAt: state.ClosedAt, PaymentID: state.PaymentID}, nil
}
