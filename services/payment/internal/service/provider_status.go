package service

import (
	"time"

	"payment/internal/midtrans"
	"payment/internal/model"
)

func providerObservation(paymentID string, verified midtrans.StatusResult, source, requestID string) model.ProviderObservation {
	observation := model.ProviderObservation{PaymentID: paymentID, Source: source, RequestID: requestID,
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
	return observation
}

// verifiedPaymentOutcome maps a server-authenticated lookup to a proposed local change.
// The repository rechecks the current attempt under a lock before applying it.
func verifiedPaymentOutcome(attempt model.PaymentAttempt, verified midtrans.StatusResult, observation model.ProviderObservation, now time.Time) model.PaymentOutcome {
	outcome := model.PaymentOutcome{TransactionID: verified.TransactionID, ProviderStatus: verified.Status,
		Observation: observation, VerifiedAt: now.UTC().Truncate(time.Microsecond)}
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
