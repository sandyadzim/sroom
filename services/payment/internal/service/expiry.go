package service

import (
	"context"
	"errors"

	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/model"
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
	assessment := dto.ExpiryAssessment{BookingID: bookingID, HoldExpiresAt: snapshot.HoldExpiresAt.UTC(),
		Result: result, ClosedAt: state.ClosedAt, PaymentID: state.PaymentID}
	if state.PaymentID == nil {
		return assessment, nil
	}
	attempt, err := s.repo.FindByID(ctx, *state.PaymentID)
	if err != nil {
		return dto.ExpiryAssessment{}, err
	}
	if !attempt.HoldExpiresAt.Equal(snapshot.HoldExpiresAt) || attempt.CustomerID != snapshot.CustomerID ||
		attempt.AmountIDR != snapshot.TotalIDR || attempt.Currency != snapshot.Currency {
		return dto.ExpiryAssessment{}, ErrNotExpirable
	}
	// A committed authorization survives retries; later paid facts are recorded for review.
	if attempt.ReleaseAuthorizedAt != nil {
		assessment.Result, assessment.ReleaseAuthorizedAt = "provider_expired", attempt.ReleaseAuthorizedAt
		return assessment, nil
	}
	if attempt.Status == "paid" || attempt.ReviewReason != nil || !s.configured || s.provider == nil {
		return assessment, nil
	}
	verified, providerError := s.provider.GetStatus(ctx, attempt.ProviderOrderID, requestID)
	observation := providerObservation(attempt.ID, verified, "status_lookup", requestID)
	outcome := verifiedPaymentOutcome(attempt, verified, observation, s.now())
	if providerError != nil {
		code := "PROVIDER_VERIFICATION_UNAVAILABLE"
		outcome = model.PaymentOutcome{VerifiedAt: s.now(), Observation: observation}
		outcome.Observation.VerificationResult, outcome.Observation.ErrorCode = "unavailable", &code
		if errors.Is(providerError, midtrans.ErrInvalidResponse) {
			code = "PROVIDER_REFERENCE_MISMATCH"
			outcome.ReviewReason = &code
			outcome.Observation.VerificationResult = "rejected"
		}
	} else {
		outcome.AuthorizeRelease = outcome.Status == "expired" && outcome.ReviewReason == nil
	}
	if err := s.repo.ApplyVerifiedObservation(ctx, attempt.ID, "", outcome); err != nil {
		return dto.ExpiryAssessment{}, err
	}
	if providerError != nil && !errors.Is(providerError, midtrans.ErrInvalidResponse) {
		return dto.ExpiryAssessment{}, ErrVerificationUnavailable
	}
	saved, err := s.repo.FindByID(ctx, attempt.ID)
	if err != nil {
		return dto.ExpiryAssessment{}, err
	}
	if saved.ReleaseAuthorizedAt != nil {
		assessment.Result, assessment.ReleaseAuthorizedAt = "provider_expired", saved.ReleaseAuthorizedAt
	}
	return assessment, nil
}
