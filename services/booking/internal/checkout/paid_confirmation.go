package checkout

import (
	"context"
	"errors"
	"time"

	"booking/internal/payment"
)

var ErrConfirmationConflict = errors.New("payment confirmation conflicts with stored facts")

const PaidConsumer = "booking.payment-events.v1"

type ConfirmationReader interface {
	Confirmation(context.Context, string, string) (payment.Confirmation, error)
}

type PaidConfirmationService struct {
	repo     *Repository
	payments ConfirmationReader
	now      func() time.Time
}

func NewPaidConfirmationService(repo *Repository, payments ConfirmationReader) *PaidConfirmationService {
	return &PaidConfirmationService{repo: repo, payments: payments, now: time.Now}
}

func (s *PaidConfirmationService) Process(ctx context.Context, body []byte) error {
	event, err := payment.DecodePaidEvent(body)
	if err != nil {
		return err
	}
	processed, err := s.repo.paidEventProcessed(ctx, event)
	if err != nil || processed {
		return err
	}
	// Verify canonical facts over HTTP before opening the Booking transaction.
	confirmed, err := s.payments.Confirmation(ctx, event.Payload.BookingID, event.CorrelationID)
	if errors.Is(err, payment.ErrNotFound) {
		return payment.ErrInvalidEvent
	}
	if err != nil {
		return err
	}
	if confirmed.Status != "paid" || confirmed.VerifiedAt == nil || confirmed.VerifiedAt.IsZero() ||
		confirmed.PaymentID != event.Payload.PaymentID || confirmed.BookingID != event.Payload.BookingID ||
		confirmed.ProviderOrderID != event.Payload.ProviderOrderID || confirmed.AmountIDR != event.Payload.AmountIDR ||
		confirmed.Currency != event.Payload.Currency || !confirmed.VerifiedAt.Equal(event.Payload.VerifiedAt) {
		return payment.ErrInvalidEvent
	}
	return s.repo.confirmVerifiedPayment(ctx, event, confirmed, s.now)
}
