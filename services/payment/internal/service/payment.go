package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrNotConfigured             = errors.New("payment integration is not configured")
	ErrNotPayable                = errors.New("booking is not payable")
	ErrHoldExpired               = errors.New("booking hold has expired")
	ErrWindowTooShort            = errors.New("not enough time remains to create a virtual account")
	ErrBillingProfileUnavailable = errors.New("verified customer billing profile is unavailable")
)

type BookingReader interface {
	PaymentContext(context.Context, string, string) (booking.PaymentContext, error)
}

type VAProvider interface {
	CreateVA(context.Context, midtrans.ChargeInput, string) (midtrans.ChargeResult, error)
}

type PaymentService struct {
	repo       *repository.PaymentRepository
	bookings   BookingReader
	provider   VAProvider
	configured bool
	now        func() time.Time
}

func NewPaymentService(repo *repository.PaymentRepository, bookings BookingReader, provider VAProvider, configured bool) *PaymentService {
	return &PaymentService{repo: repo, bookings: bookings, provider: provider, configured: configured, now: time.Now}
}

func (s *PaymentService) CreateVA(ctx context.Context, input dto.CreateVARequest, requestID string) (dto.VAResponse, bool, error) {
	if !s.configured {
		return dto.VAResponse{}, false, ErrNotConfigured
	}
	snapshot, err := s.bookings.PaymentContext(ctx, input.BookingID, requestID)
	if err != nil {
		return dto.VAResponse{}, false, err
	}
	customerID, customerError := uuid.Parse(snapshot.CustomerID)
	bookingID, bookingError := uuid.Parse(snapshot.BookingID)
	if customerError != nil || bookingError != nil || snapshot.TotalIDR <= 0 || snapshot.Currency != "IDR" || snapshot.HoldExpiresAt.IsZero() {
		return dto.VAResponse{}, false, booking.ErrUnavailable
	}
	if customerID.String() != input.CustomerID || bookingID.String() != input.BookingID {
		return dto.VAResponse{}, false, booking.ErrNotFound
	}
	if snapshot.Status != "pending_payment" {
		return dto.VAResponse{}, false, ErrNotPayable
	}
	if !s.now().Before(snapshot.HoldExpiresAt) {
		return dto.VAResponse{}, false, ErrHoldExpired
	}
	address, err := mail.ParseAddress(input.CustomerEmail)
	if err != nil || address.Address != input.CustomerEmail {
		return dto.VAResponse{}, false, ErrBillingProfileUnavailable
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return dto.VAResponse{}, false, repository.ErrUnavailable
	}
	attempt := model.PaymentAttempt{ID: id.String(), BookingID: input.BookingID, CustomerID: input.CustomerID,
		CustomerFullName: &input.CustomerFullName, CustomerEmail: &input.CustomerEmail,
		ProviderOrderID: "sroom-" + id.String(), AmountIDR: snapshot.TotalIDR, Currency: snapshot.Currency,
		HoldExpiresAt: snapshot.HoldExpiresAt.UTC().Truncate(time.Microsecond)}
	fingerprint := sha256.Sum256([]byte(input.BookingID + "|bni"))
	attempt, created, err := s.repo.Reserve(ctx, attempt, input.IdempotencyKey, hex.EncodeToString(fingerprint[:]))
	if err != nil {
		return dto.VAResponse{}, false, err
	}
	if !created {
		return s.response(attempt), false, nil
	}
	// Check immediately before the provider call; the original deadline never moves.
	if attempt.HoldExpiresAt.Truncate(time.Second).Sub(s.now()) < 20*time.Second {
		reason := "PAYMENT_WINDOW_TOO_SHORT"
		attempt.ReviewReason = &reason
		_, err := s.repo.SaveCharge(ctx, attempt, model.ProviderObservation{PaymentID: attempt.ID,
			VerificationResult: "rejected", ErrorCode: &reason, RequestID: requestID})
		if err != nil {
			return dto.VAResponse{}, false, err
		}
		return dto.VAResponse{}, false, ErrWindowTooShort
	}
	result, providerError := s.provider.CreateVA(ctx, midtrans.ChargeInput{
		OrderID: attempt.ProviderOrderID, AmountIDR: attempt.AmountIDR, HoldExpiresAt: attempt.HoldExpiresAt}, requestID)
	observation := model.ProviderObservation{PaymentID: attempt.ID, VerificationResult: "verified", RequestID: requestID}
	if result.Digest != "" {
		observation.PayloadDigest = &result.Digest
	}
	if result.TransactionID != "" {
		observation.ProviderTransactionID = &result.TransactionID
	}
	if result.Status != "" {
		observation.ProviderStatus = &result.Status
	}
	if result.Currency != "" {
		observation.Currency = &result.Currency
	}
	if result.AmountIDR > 0 {
		observation.AmountIDR = &result.AmountIDR
	}
	if !result.ExpiresAt.IsZero() {
		expires := result.ExpiresAt.UTC()
		attempt.ProviderExpiresAt = &expires
	}
	if providerError != nil {
		code := "PROVIDER_UNAVAILABLE"
		observation.VerificationResult = "unavailable"
		if errors.Is(providerError, midtrans.ErrInvalidResponse) {
			code = "PROVIDER_RESPONSE_MISMATCH"
			observation.VerificationResult = "rejected"
			attempt.ReviewReason = &code
		}
		observation.ErrorCode = &code
	} else {
		attempt.Status, attempt.VANumber = "pending", &result.VANumber
		attempt.ProviderTransactionID, attempt.ProviderStatus = &result.TransactionID, &result.Status
		if !s.now().Before(result.ExpiresAt) {
			code := "PROVIDER_RESPONSE_EXPIRED"
			attempt.Status, attempt.ReviewReason = "initiating", &code
			observation.VerificationResult, observation.ErrorCode = "rejected", &code
		}
	}
	saved, err := s.repo.SaveCharge(ctx, attempt, observation)
	if err != nil {
		return dto.VAResponse{}, false, err
	}
	return s.response(saved), true, nil
}

func (s *PaymentService) response(attempt model.PaymentAttempt) dto.VAResponse {
	result := dto.VAResponse{PaymentID: attempt.ID, BookingID: attempt.BookingID, Status: attempt.Status,
		AmountIDR: attempt.AmountIDR, Currency: attempt.Currency, Bank: attempt.Bank,
		HoldExpiresAt: attempt.HoldExpiresAt.UTC(), ReviewRequired: attempt.ReviewReason != nil}
	if attempt.ProviderExpiresAt != nil {
		expires := attempt.ProviderExpiresAt.UTC()
		result.ProviderExpiresAt = &expires
		if attempt.Status == "pending" && !result.ReviewRequired && s.now().Before(expires) && s.now().Before(attempt.HoldExpiresAt) {
			result.VANumber = attempt.VANumber
		}
	}
	return result
}
