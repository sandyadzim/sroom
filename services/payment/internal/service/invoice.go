package service

import (
	"context"
	"errors"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/model"
	"payment/internal/repository"
)

var ErrInvoiceNotAvailable = errors.New("invoice is not available yet")

type InvoiceBookingReader interface {
	BookingReader
	InvoiceContext(context.Context, string, string) (booking.InvoiceContext, error)
}

type InvoiceService struct {
	repo     *repository.PaymentRepository
	bookings InvoiceBookingReader
	now      func() time.Time
	wita     *time.Location
}

func NewInvoiceService(repo *repository.PaymentRepository, bookings InvoiceBookingReader) (*InvoiceService, error) {
	wita, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		return nil, errors.New("could not load Asia/Makassar time zone")
	}
	return &InvoiceService{repo: repo, bookings: bookings, now: time.Now, wita: wita}, nil
}

func (s *InvoiceService) Process(ctx context.Context, body []byte) error {
	event, err := model.DecodeConfirmedEvent(body)
	if err != nil {
		return err
	}
	processed, err := s.repo.InvoiceEventProcessed(ctx, event)
	if err != nil || processed {
		return err
	}
	// HTTP runs before the invoice/inbox transaction; no token or profile enters events.
	snapshot, err := s.bookings.InvoiceContext(ctx, event.Payload.BookingID, event.CorrelationID)
	if err != nil {
		return err
	}
	if snapshot.BookingID != event.Payload.BookingID || snapshot.PaymentID != event.Payload.PaymentID || snapshot.CustomerID != event.Payload.CustomerID || snapshot.Status != "confirmed" ||
		snapshot.TotalIDR != event.Payload.TotalIDR || snapshot.Currency != event.Payload.Currency || !snapshot.ConfirmedAt.Equal(event.Payload.ConfirmedAt) || snapshot.ConfirmationEventID != event.EventID {
		return repository.ErrInvoiceConflict
	}
	return s.repo.IssueInvoice(ctx, event, snapshot, s.now)
}

func (s *InvoiceService) Read(ctx context.Context, bookingID, customerID, requestID string) (dto.InvoiceResponse, error) {
	invoice, err := s.repo.FindInvoice(ctx, bookingID)
	if err == nil {
		if customerID != "" && invoice.CustomerID != customerID {
			return dto.InvoiceResponse{}, booking.ErrNotFound
		}
		return s.response(invoice), nil
	}
	if !errors.Is(err, repository.ErrInvoiceNotFound) {
		return dto.InvoiceResponse{}, err
	}
	// Missing invoice is different from an inaccessible/missing booking.
	snapshot, err := s.bookings.PaymentContext(ctx, bookingID, requestID)
	if err != nil {
		return dto.InvoiceResponse{}, err
	}
	if snapshot.BookingID != bookingID || snapshot.CustomerID == "" {
		return dto.InvoiceResponse{}, booking.ErrUnavailable
	}
	if customerID != "" && snapshot.CustomerID != customerID {
		return dto.InvoiceResponse{}, booking.ErrNotFound
	}
	return dto.InvoiceResponse{}, ErrInvoiceNotAvailable
}

func (s *InvoiceService) response(invoice model.Invoice) dto.InvoiceResponse {
	return dto.InvoiceResponse{ID: invoice.ID, InvoiceNumber: invoice.InvoiceNumber, BookingID: invoice.BookingID, PaymentID: invoice.PaymentID, CustomerID: invoice.CustomerID,
		ProviderOrderID: invoice.ProviderOrderID, BilledTo: dto.InvoiceCustomer{FullName: invoice.BilledToName, Email: invoice.BilledToEmail},
		VariantID: invoice.VariantID, VehicleLabel: invoice.VehicleLabel, PickupAt: invoice.PickupAt.In(s.wita), ReturnAt: invoice.ReturnAt.In(s.wita),
		RentalDays: invoice.RentalDays, DailyRateIDR: invoice.DailyRateIDR, TotalIDR: invoice.TotalIDR, Currency: invoice.Currency, Bank: invoice.Bank,
		PaymentMethod: "virtual_account", Status: "paid", DocumentType: "non_tax_invoice", ConfirmedAt: invoice.ConfirmedAt.UTC(), IssuedAt: invoice.IssuedAt.UTC()}
}
