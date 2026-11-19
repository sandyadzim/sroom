package service

import (
	"context"
	"errors"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
)

func (s *PaymentService) CustomerPayment(ctx context.Context, bookingID, customerID, requestID string) (dto.CustomerPaymentResponse, error) {
	snapshot, attempt, err := s.ownedPayment(ctx, bookingID, customerID, requestID)
	if err != nil {
		return dto.CustomerPaymentResponse{}, err
	}
	return s.customerPaymentResponse(snapshot, attempt), nil
}

// GET and refresh use the same ownership checks before reading provider facts.
func (s *PaymentService) ownedPayment(ctx context.Context, bookingID, customerID, requestID string) (booking.PaymentContext, model.PaymentAttempt, error) {
	snapshot, err := s.readBooking(ctx, bookingID, requestID)
	if err != nil {
		return snapshot, model.PaymentAttempt{}, err
	}
	if snapshot.CustomerID != customerID {
		return snapshot, model.PaymentAttempt{}, booking.ErrNotFound
	}
	attempt, err := s.repo.FindByBooking(ctx, bookingID)
	if errors.Is(err, repository.ErrPaymentNotFound) {
		return snapshot, model.PaymentAttempt{}, nil
	}
	if err != nil {
		return snapshot, attempt, err
	}
	if attempt.CustomerID != customerID {
		return snapshot, attempt, booking.ErrNotFound
	}
	return snapshot, attempt, nil
}

func (s *PaymentService) customerPaymentResponse(snapshot booking.PaymentContext, attempt model.PaymentAttempt) dto.CustomerPaymentResponse {
	if attempt.ID == "" {
		return dto.CustomerPaymentResponse{BookingID: snapshot.BookingID, BookingStatus: snapshot.Status, Status: "not_started",
			AmountIDR: snapshot.TotalIDR, Currency: snapshot.Currency, HoldExpiresAt: snapshot.HoldExpiresAt.UTC()}
	}
	instructions := s.response(attempt)
	return dto.CustomerPaymentResponse{PaymentID: &attempt.ID, BookingID: attempt.BookingID, BookingStatus: snapshot.Status,
		Status: attempt.Status, AmountIDR: attempt.AmountIDR, Currency: attempt.Currency, Bank: &attempt.Bank,
		VANumber: instructions.VANumber, HoldExpiresAt: instructions.HoldExpiresAt, ProviderExpiresAt: instructions.ProviderExpiresAt,
		VerifiedAt: attempt.VerifiedAt, ProviderPaidAt: attempt.ProviderPaidAt, ReviewRequired: instructions.ReviewRequired}
}

func (s *PaymentService) ListPayments(ctx context.Context, query dto.ListPaymentsQuery) ([]dto.StaffPaymentResponse, int64, error) {
	attempts, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	result := make([]dto.StaffPaymentResponse, 0, len(attempts))
	for _, attempt := range attempts {
		result = append(result, staffPayment(attempt))
	}
	return result, total, nil
}

func (s *PaymentService) PaymentDetail(ctx context.Context, id, requestID string) (dto.StaffPaymentDetail, error) {
	attempt, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return dto.StaffPaymentDetail{}, err
	}
	snapshot, err := s.readBooking(ctx, attempt.BookingID, requestID)
	if err != nil || snapshot.CustomerID != attempt.CustomerID {
		return dto.StaffPaymentDetail{}, booking.ErrUnavailable
	}
	observations, err := s.repo.Observations(ctx, id)
	if err != nil {
		return dto.StaffPaymentDetail{}, err
	}
	timeline := make([]dto.PaymentTimelineEntry, 0, len(observations)+1)
	timeline = append(timeline, dto.PaymentTimelineEntry{ID: attempt.ID, Source: "created", OccurredAt: attempt.CreatedAt.UTC()})
	for _, observation := range observations {
		timeline = append(timeline, dto.PaymentTimelineEntry{ID: observation.ID, Source: observation.Source,
			OccurredAt: observation.ObservedAt.UTC(), ProviderStatus: observation.ProviderStatus,
			VerificationResult: &observation.VerificationResult, ErrorCode: observation.ErrorCode})
	}
	return dto.StaffPaymentDetail{StaffPaymentResponse: staffPayment(attempt), BookingStatus: snapshot.Status,
		ProviderStatus: attempt.ProviderStatus, HoldExpiresAt: attempt.HoldExpiresAt.UTC(), ProviderExpiresAt: attempt.ProviderExpiresAt,
		VerifiedAt: attempt.VerifiedAt, ProviderPaidAt: attempt.ProviderPaidAt, Timeline: timeline}, nil
}

func staffPayment(attempt model.PaymentAttempt) dto.StaffPaymentResponse {
	return dto.StaffPaymentResponse{PaymentID: attempt.ID, BookingID: attempt.BookingID, CustomerID: attempt.CustomerID,
		Provider: attempt.Provider, ProviderOrderID: attempt.ProviderOrderID, ProviderTransactionID: attempt.ProviderTransactionID,
		Status: attempt.Status, Bank: attempt.Bank, AmountIDR: attempt.AmountIDR, Currency: attempt.Currency,
		ReviewRequired: attempt.ReviewReason != nil, ReviewReason: attempt.ReviewReason,
		CreatedAt: attempt.CreatedAt.UTC(), UpdatedAt: attempt.UpdatedAt.UTC()}
}

func (s *PaymentService) readBooking(ctx context.Context, bookingID, requestID string) (booking.PaymentContext, error) {
	snapshot, err := s.bookings.PaymentContext(ctx, bookingID, requestID)
	if err != nil {
		return snapshot, err
	}
	id, idError := uuid.Parse(snapshot.BookingID)
	customerID, customerError := uuid.Parse(snapshot.CustomerID)
	if idError != nil || customerError != nil || id.String() != bookingID || customerID == uuid.Nil ||
		snapshot.TotalIDR <= 0 || snapshot.Currency != "IDR" || snapshot.HoldExpiresAt.IsZero() || snapshot.Status == "" {
		return snapshot, booking.ErrUnavailable
	}
	snapshot.CustomerID = customerID.String()
	return snapshot, nil
}
