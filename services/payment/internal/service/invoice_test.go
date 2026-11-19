package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type invoiceBookingReader struct {
	*testBookingReader
	invoice      booking.InvoiceContext
	invoiceError error
	mu           sync.Mutex
	calls        int
}

func (r *invoiceBookingReader) InvoiceContext(context.Context, string, string) (booking.InvoiceContext, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return r.invoice, r.invoiceError
}

type invoiceFixture struct {
	service  *InvoiceService
	payments *PaymentService
	db       *gorm.DB
	reader   *invoiceBookingReader
	event    model.ConfirmedEvent
}

func newInvoiceFixture(t *testing.T) invoiceFixture {
	t.Helper()
	payments, db, reader, _, input := paymentFixture(t)
	at := payments.now()
	result, _, err := payments.CreateVA(t.Context(), input, "invoice-create")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE payment.payment_attempts SET status = 'paid',verified_at = ? WHERE id = ?", at, result.PaymentID).Error; err != nil {
		t.Fatal(err)
	}
	confirmed := at.Add(time.Minute)
	event := model.ConfirmedEvent{EventID: uuid.NewString(), EventType: model.ConfirmedEventType, SchemaVersion: 1, OccurredAt: confirmed, AggregateID: input.BookingID, CorrelationID: "invoice-test", CausationID: uuid.NewString(),
		Payload: model.ConfirmedPayload{BookingID: input.BookingID, PaymentID: result.PaymentID, CustomerID: input.CustomerID, ConfirmedAt: confirmed, TotalIDR: 150000, Currency: "IDR"}}
	profile := &invoiceBookingReader{testBookingReader: reader, invoice: booking.InvoiceContext{BookingID: input.BookingID, CustomerID: input.CustomerID, VariantID: uuid.NewString(), VehicleLabel: "Synthetic car", Status: "confirmed", PaymentID: result.PaymentID,
		ProviderOrderID: "sroom-" + result.PaymentID, ConfirmationEventID: event.EventID, ConfirmedAt: confirmed, PickupAt: at.Add(24 * time.Hour), ReturnAt: at.Add(26 * time.Hour), RentalDays: 1, DailyRateIDR: 150000, TotalIDR: 150000, Currency: "IDR"}}
	service, err := NewInvoiceService(repository.NewPaymentRepository(db), profile)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return confirmed.Add(time.Second) }
	return invoiceFixture{service: service, payments: payments, db: db, reader: profile, event: event}
}

func confirmedBody(t *testing.T, event model.ConfirmedEvent) []byte {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestInvoiceIssueSnapshotRetryAndReadsPostgres(t *testing.T) {
	f := newInvoiceFixture(t)
	body := confirmedBody(t, f.event)
	if err := f.service.Process(t.Context(), body); err != nil {
		t.Fatal(err)
	}
	first, err := f.service.Read(t.Context(), f.event.AggregateID, f.event.Payload.CustomerID, "invoice-read")
	if err != nil || first.InvoiceNumber != "INV-"+strings.ToUpper(first.ID) || first.TotalIDR != 150000 || first.BilledTo.FullName != "Synthetic Customer" || first.BilledTo.Email != "customer@example.test" || first.DocumentType != "non_tax_invoice" {
		t.Fatal("invoice did not preserve trusted snapshots")
	}
	if _, offset := first.PickupAt.Zone(); offset != 8*60*60 {
		t.Fatal("schedule is not WITA")
	}
	// Current profile/rate data changes never rewrite the issued snapshot.
	if err := f.db.Exec("UPDATE payment.payment_attempts SET customer_full_name = 'Changed',customer_email = 'changed@example.test' WHERE id = ?", first.PaymentID).Error; err != nil {
		t.Fatal(err)
	}
	f.reader.invoice.VehicleLabel = "Changed car"
	f.reader.invoiceError, f.reader.err = booking.ErrUnavailable, booking.ErrUnavailable
	restarted, err := NewInvoiceService(repository.NewPaymentRepository(f.db), f.reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Process(t.Context(), body); err != nil {
		t.Fatal("committed inbox still depended on Booking")
	}
	again, err := restarted.Read(t.Context(), first.BookingID, first.CustomerID, "retry")
	if err != nil || !sameInvoice(first, again) {
		t.Fatal("read/restart rewrote invoice or required Booking")
	}
	if _, err := restarted.Read(t.Context(), first.BookingID, uuid.NewString(), "other"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal("invoice disclosed to another customer")
	}
	if _, err := restarted.Read(t.Context(), first.BookingID, "", "staff"); err != nil {
		t.Fatal(err)
	}
	if countPaymentRows(t, f.db, "invoices", "booking_id", first.BookingID) != 1 || countPaymentRows(t, f.db, "inbox_events", "aggregate_id", first.BookingID) != 1 {
		t.Fatal("duplicate repeated invoice effects")
	}
}

func TestInvoiceRejectsAndRollsBackPostgres(t *testing.T) {
	for _, name := range []string{"pending", "review", "expired", "legacy", "wrong reference", "wrong amount", "not confirmed", "unavailable", "rollback"} {
		t.Run(name, func(t *testing.T) {
			f := newInvoiceFixture(t)
			want := repository.ErrInvoiceConflict
			switch name {
			case "pending", "expired":
				if err := f.db.Exec("UPDATE payment.payment_attempts SET status = ? WHERE id = ?", name, f.event.Payload.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
			case "review":
				if err := f.db.Exec("UPDATE payment.payment_attempts SET review_reason = 'SYNTHETIC_REVIEW' WHERE id = ?", f.event.Payload.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
			case "legacy":
				if err := f.db.Exec("UPDATE payment.payment_attempts SET customer_email = NULL,customer_full_name = NULL WHERE id = ?", f.event.Payload.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
				want = repository.ErrInvoiceProfileMissing
			case "wrong reference":
				f.reader.invoice.PaymentID = uuid.NewString()
			case "wrong amount":
				f.reader.invoice.TotalIDR++
			case "not confirmed":
				f.reader.invoice.Status = "payment_review"
			case "unavailable":
				f.reader.invoiceError = booking.ErrUnavailable
				want = booking.ErrUnavailable
			case "rollback":
				f.reader.invoice.RentalDays = 0
				want = repository.ErrUnavailable
			}
			if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); !errors.Is(err, want) {
				t.Fatalf("error=%v want=%v", err, want)
			}
			if countPaymentRows(t, f.db, "invoices", "booking_id", f.event.AggregateID) != 0 || countPaymentRows(t, f.db, "inbox_events", "aggregate_id", f.event.AggregateID) != 0 {
				t.Fatal("failure committed partial invoice/inbox")
			}
			if name == "unavailable" {
				f.reader.invoiceError = nil
				if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); err != nil {
					t.Fatal("retry did not recover")
				}
			}
		})
	}
}

func TestInvoiceConcurrentAndConflictingEventsPostgres(t *testing.T) {
	f := newInvoiceFixture(t)
	body := confirmedBody(t, f.event)
	var wg sync.WaitGroup
	results := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- f.service.Process(t.Context(), body) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if countPaymentRows(t, f.db, "invoices", "booking_id", f.event.AggregateID) != 1 || countPaymentRows(t, f.db, "inbox_events", "aggregate_id", f.event.AggregateID) != 1 {
		t.Fatal("concurrent delivery duplicated invoice")
	}
	f.event.Payload.TotalIDR++
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); !errors.Is(err, repository.ErrInvoiceConflict) {
		t.Fatal("changed payload reused event ID")
	}
	f.event.Payload.TotalIDR--
	f.event.EventID = uuid.NewString()
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); !errors.Is(err, repository.ErrInvoiceConflict) {
		t.Fatal("noncanonical confirmation event accepted")
	}
	// Even a second canonical envelope for the same confirmed booking cannot issue twice.
	f.reader.invoice.ConfirmationEventID = f.event.EventID
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	if countPaymentRows(t, f.db, "invoices", "booking_id", f.event.AggregateID) != 1 {
		t.Fatal("different event ID created a second invoice")
	}
}

func TestInvoiceUnavailableAndOwnershipPostgres(t *testing.T) {
	f := newInvoiceFixture(t)
	if _, err := f.service.Read(t.Context(), f.event.AggregateID, f.event.Payload.CustomerID, "waiting"); !errors.Is(err, ErrInvoiceNotAvailable) {
		t.Fatal("missing invoice was not distinguished")
	}
	if _, err := f.service.Read(t.Context(), f.event.AggregateID, uuid.NewString(), "other"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal("waiting invoice disclosed ownership")
	}
	f.reader.err = booking.ErrNotFound
	if _, err := f.service.Read(t.Context(), f.event.AggregateID, "", "missing"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestBillingProfileFirstAttemptWinsPostgres(t *testing.T) {
	svc, db, _, provider, input := paymentFixture(t)
	first, _, err := svc.CreateVA(t.Context(), input, "first")
	if err != nil {
		t.Fatal(err)
	}
	input.CustomerFullName, input.CustomerEmail = "Changed", "changed@example.test"
	if _, _, err := svc.CreateVA(t.Context(), input, "retry"); err != nil {
		t.Fatal(err)
	}
	saved := reloadAttempt(t, db, first.PaymentID)
	if saved.CustomerFullName == nil || *saved.CustomerFullName != "Synthetic Customer" || saved.CustomerEmail == nil || *saved.CustomerEmail != "customer@example.test" || provider.calls != 1 {
		t.Fatal("retry replaced the first verified profile")
	}
	input.BookingID = uuid.NewString()
	input.CustomerEmail = ""
	// Eligibility/ownership errors still precede profile validation.
	if _, _, err := svc.CreateVA(t.Context(), input, "forged"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal(err)
	}
}

func sameInvoice(a, b dto.InvoiceResponse) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}

func TestInvoiceAllowsEmptyNamePostgres(t *testing.T) {
	f := newInvoiceFixture(t)
	if err := f.db.Exec("UPDATE payment.payment_attempts SET customer_full_name = '' WHERE id = ?", f.event.Payload.PaymentID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	result, err := f.service.Read(t.Context(), f.event.AggregateID, f.event.Payload.CustomerID, "empty-name")
	if err != nil || result.BilledTo.FullName != "" || result.BilledTo.Email != "customer@example.test" {
		t.Fatal("empty name must preserve verified email for display fallback")
	}
}
