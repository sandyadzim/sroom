package checkout

import "testing"

func TestVerifiedInvoiceContextPostgres(t *testing.T) {
	f := paidFixture(t)
	if err := f.service.Process(t.Context(), eventBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(nil, NewRepository(f.db))
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.InvoiceContext(t.Context(), f.event.Payload.BookingID)
	if err != nil || result.Status != "confirmed" || result.PaymentID != f.reader.data.PaymentID || result.ProviderOrderID != f.reader.data.ProviderOrderID || result.ConfirmationEventID == "" || result.TotalIDR != 300000 || result.RentalDays != 2 || result.DailyRateIDR != 150000 {
		t.Fatal("canonical invoice context lost its committed snapshot")
	}
	if _, offset := result.PickupAt.Zone(); offset != 8*60*60 {
		t.Fatal("invoice schedule did not use WITA")
	}
	if result.ConfirmationEventID == f.event.EventID {
		t.Fatal("context returned the paid event instead of the confirmed event")
	}
}
