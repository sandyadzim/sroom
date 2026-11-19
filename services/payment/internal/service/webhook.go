package service

import (
	"context"
	"errors"
	"time"

	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/repository"
)

var (
	ErrInvalidSignature        = errors.New("invalid notification signature")
	ErrVerificationUnavailable = errors.New("could not verify payment status")
)

type StatusProvider interface {
	GetStatus(context.Context, string, string) (midtrans.StatusResult, error)
}

type WebhookService struct {
	repo     *repository.PaymentRepository
	provider StatusProvider
	key      string
	now      func() time.Time
}

func NewWebhookService(repo *repository.PaymentRepository, provider StatusProvider, key string) *WebhookService {
	return &WebhookService{repo: repo, provider: provider, key: key, now: time.Now}
}

func (s *WebhookService) Process(ctx context.Context, notification dto.MidtransNotification, hash, requestID string) error {
	if s.key == "" {
		return ErrNotConfigured
	}
	if !midtrans.VerifySignature(notification.OrderID, notification.StatusCode, notification.GrossAmount, notification.SignatureKey, s.key) {
		return ErrInvalidSignature
	}
	attempt, receipt, done, err := s.repo.ReceiveWebhook(ctx, notification.OrderID, hash)
	if err != nil || done {
		return err
	}
	verified, providerError := s.provider.GetStatus(ctx, attempt.ProviderOrderID, requestID)
	observation := providerObservation(attempt.ID, verified, "webhook", requestID)
	if providerError != nil && !errors.Is(providerError, midtrans.ErrInvalidResponse) {
		code := "PROVIDER_VERIFICATION_UNAVAILABLE"
		observation.VerificationResult, observation.ErrorCode = "unavailable", &code
		if err := s.repo.WebhookUnavailable(ctx, receipt.ID, observation); err != nil {
			return err
		}
		return ErrVerificationUnavailable
	}
	outcome := verifiedPaymentOutcome(attempt, verified, observation, s.now())
	amount, amountError := midtrans.ParseAmount(notification.GrossAmount)
	if providerError != nil || amountError != nil || amount != attempt.AmountIDR ||
		(notification.TransactionID != "" && notification.TransactionID != verified.TransactionID) ||
		(notification.MerchantID != "" && notification.MerchantID != verified.MerchantID) ||
		(notification.Currency != "" && notification.Currency != attempt.Currency) ||
		(notification.PaymentType != "" && notification.PaymentType != "bank_transfer") {
		reason := "PROVIDER_REFERENCE_MISMATCH"
		outcome.Status, outcome.ReviewReason = "", &reason
		outcome.Observation.VerificationResult, outcome.Observation.ErrorCode = "rejected", &reason
	}
	return s.repo.ApplyVerifiedObservation(ctx, attempt.ID, receipt.ID, outcome)
}
