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
	observation := model.ProviderObservation{PaymentID: attempt.ID, Source: "webhook", RequestID: requestID,
		VerificationResult: "verified"}
	if verified.Digest != "" {
		observation.PayloadDigest = &verified.Digest
	}
	if verified.TransactionID != "" {
		observation.ProviderTransactionID = &verified.TransactionID
	}
	if verified.Status != "" {
		observation.ProviderStatus = &verified.Status
	}
	if verified.AmountIDR > 0 {
		observation.AmountIDR = &verified.AmountIDR
	}
	if verified.Currency != "" {
		observation.Currency = &verified.Currency
	}
	if providerError != nil && !errors.Is(providerError, midtrans.ErrInvalidResponse) {
		code := "PROVIDER_VERIFICATION_UNAVAILABLE"
		observation.VerificationResult, observation.ErrorCode = "unavailable", &code
		if err := s.repo.WebhookUnavailable(ctx, receipt.ID, observation); err != nil {
			return err
		}
		return ErrVerificationUnavailable
	}
	outcome := s.VerifiedObservation(attempt, verified, observation)
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

// VerifiedObservation maps a server-authenticated lookup to a proposed local change.
// The repository rechecks the current attempt under a lock before applying it.
func (s *WebhookService) VerifiedObservation(attempt model.PaymentAttempt, verified midtrans.StatusResult, observation model.ProviderObservation) model.PaymentOutcome {
	outcome := model.PaymentOutcome{TransactionID: verified.TransactionID, ProviderStatus: verified.Status,
		Observation: observation, VerifiedAt: s.now().UTC().Truncate(time.Microsecond)}
	if verified.OrderID != attempt.ProviderOrderID || verified.TransactionID == "" || verified.MerchantID == "" ||
		verified.AmountIDR != attempt.AmountIDR || verified.Currency != attempt.Currency ||
		verified.PaymentType != "bank_transfer" || verified.Bank != attempt.Bank ||
		(attempt.ProviderTransactionID != nil && *attempt.ProviderTransactionID != verified.TransactionID) {
		reason := "PROVIDER_REFERENCE_MISMATCH"
		outcome.ReviewReason = &reason
	} else {
		switch verified.Status {
		case "pending":
			outcome.Status = "pending"
		case "settlement":
			if verified.StatusCode == "200" && verified.FraudStatus == "accept" {
				outcome.Status = "paid"
			}
		case "expire":
			outcome.Status = "expired"
		case "deny", "cancel", "failure":
			outcome.Status = "failed"
		}
		if outcome.Status == "" {
			reason := "PROVIDER_STATUS_UNRECOGNIZED"
			outcome.ReviewReason = &reason
		}
	}
	if outcome.ReviewReason != nil {
		outcome.Observation.VerificationResult = "rejected"
		outcome.Observation.ErrorCode = outcome.ReviewReason
	}
	return outcome
}
