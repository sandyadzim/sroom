package service

import (
	"context"
	"payment/internal/dto"
)

func (s *PaymentService) Confirmation(ctx context.Context, bookingID string) (dto.PaymentConfirmation, error) {
	attempt, err := s.repo.FindByBooking(ctx, bookingID)
	if err != nil {
		return dto.PaymentConfirmation{}, err
	}
	return dto.PaymentConfirmation{PaymentID: attempt.ID, BookingID: attempt.BookingID, CustomerID: attempt.CustomerID,
		ProviderOrderID: attempt.ProviderOrderID, AmountIDR: attempt.AmountIDR, Currency: attempt.Currency, Status: attempt.Status,
		HoldExpiresAt: attempt.HoldExpiresAt.UTC(), VerifiedAt: attempt.VerifiedAt, ProviderPaidAt: attempt.ProviderPaidAt,
		ReviewRequired: attempt.ReviewReason != nil}, nil
}
