package service

import (
	"context"
	"errors"
	"time"

	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/model"
	"payment/internal/repository"
)

type RefreshPaymentService struct {
	payments   *PaymentService
	provider   StatusProvider
	configured bool
}

func NewRefreshPaymentService(payments *PaymentService, provider StatusProvider, configured bool) *RefreshPaymentService {
	return &RefreshPaymentService{payments: payments, provider: provider, configured: configured}
}

func (s *RefreshPaymentService) Refresh(ctx context.Context, bookingID, customerID, requestID string) (dto.CustomerPaymentResponse, error) {
	snapshot, attempt, err := s.payments.ownedPayment(ctx, bookingID, customerID, requestID)
	if err != nil {
		return dto.CustomerPaymentResponse{}, err
	}
	if attempt.ID == "" || attempt.Status == "paid" {
		return s.payments.customerPaymentResponse(snapshot, attempt), nil
	}
	if !s.configured {
		return dto.CustomerPaymentResponse{}, ErrNotConfigured
	}

	verified, providerError := s.provider.GetStatus(ctx, attempt.ProviderOrderID, requestID)
	observation := providerObservation(attempt.ID, verified, "status_lookup", requestID)
	if providerError != nil && !errors.Is(providerError, midtrans.ErrInvalidResponse) {
		code := "PROVIDER_VERIFICATION_UNAVAILABLE"
		observation.VerificationResult, observation.ErrorCode = "unavailable", &code
		outcome := model.PaymentOutcome{Observation: observation, VerifiedAt: s.payments.now().UTC().Truncate(time.Microsecond)}
		if err := s.payments.repo.ApplyVerifiedObservation(ctx, attempt.ID, "", outcome); err != nil {
			return dto.CustomerPaymentResponse{}, err
		}
		return dto.CustomerPaymentResponse{}, ErrVerificationUnavailable
	}
	outcome := verifiedPaymentOutcome(attempt, verified, observation, s.payments.now())
	if providerError != nil {
		reason := "PROVIDER_REFERENCE_MISMATCH"
		outcome.Status, outcome.ReviewReason = "", &reason
		outcome.Observation.VerificationResult, outcome.Observation.ErrorCode = "rejected", &reason
	}
	if err := s.payments.repo.ApplyVerifiedObservation(ctx, attempt.ID, "", outcome); err != nil {
		return dto.CustomerPaymentResponse{}, err
	}
	saved, err := s.payments.repo.FindByID(ctx, attempt.ID)
	if err != nil {
		return dto.CustomerPaymentResponse{}, repository.ErrUnavailable
	}
	return s.payments.customerPaymentResponse(snapshot, saved), nil
}
