package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking/internal/checkout"
	"booking/internal/payment"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestHoldToVerifiedConfirmationHTTPPostgres(t *testing.T) {
	pool := openAllocationTestDB(t)
	cleanupTestBookings(t, pool, testCustomerID)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, query := range []string{
			"DELETE FROM booking.inbox_events WHERE booking_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.outbox_events WHERE aggregate_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.booking_status_history WHERE booking_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.idempotency_records WHERE customer_id = $1",
		} {
			if _, err := pool.ExecContext(ctx, query, testCustomerID); err != nil {
				t.Error("could not clean flow fixture")
			}
		}
	})
	router := quoteTestRouter(t, quoteTestCatalog(t, 200, "active", []string{uuid.NewString()}), pool)
	payload := holdRequest{quoteRequest: quoteRequest{VariantID: quoteVariantID, PickupAt: pickup, ReturnAt: nextReturn}, ExpectedTotalIDR: 1350000, TermsVersion: "rental-terms-v1", AcceptTerms: true}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", customerBookingsPath, strings.NewReader(string(body)))
	request.Header.Set("Authorization", "Bearer CUSTOMER")
	request.Header.Set("X-Request-ID", quoteRequestID)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 201 {
		t.Fatalf("hold=%d %s", response.Code, response.Body.String())
	}
	var hold holdTestResponse
	if err := json.Unmarshal(response.Body.Bytes(), &hold); err != nil {
		t.Fatal(err)
	}
	paymentID := uuid.NewString()
	verified := hold.Data.HoldExpiresAt.Add(-time.Minute)
	canonical := payment.Confirmation{PaymentID: paymentID, BookingID: hold.Data.ID, CustomerID: testCustomerID, ProviderOrderID: "sroom-" + paymentID, AmountIDR: hold.Data.TotalIDR, Currency: "IDR", Status: "paid", HoldExpiresAt: hold.Data.HoldExpiresAt, VerifiedAt: &verified}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/bookings/"+hold.Data.ID+"/payment-confirmation" || r.Header.Get("Authorization") != "Bearer synthetic-pair" || r.Header.Get("X-Request-ID") != quoteRequestID {
			t.Error("incorrect Payment verification")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": canonical})
	}))
	defer server.Close()
	client, err := payment.NewClient(server.URL, "synthetic-pair")
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	service := checkout.NewPaidConfirmationService(checkout.NewRepository(db), client)
	event := payment.PaidEvent{EventID: uuid.NewString(), EventType: payment.PaidEventType, SchemaVersion: 1, OccurredAt: verified, AggregateID: paymentID, CorrelationID: quoteRequestID, CausationID: json.RawMessage("null"),
		Payload: payment.PaidPayload{PaymentID: paymentID, BookingID: hold.Data.ID, ProviderOrderID: canonical.ProviderOrderID, AmountIDR: hold.Data.TotalIDR, Currency: "IDR", VerifiedAt: verified, ProviderPaidAt: json.RawMessage("null")}}
	body, err = json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Process(t.Context(), body); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ token, path string }{{"CUSTOMER", customerBookingsPath + "/" + hold.Data.ID}, {"ADMIN", staffBookingsPath + "/" + hold.Data.ID}} {
		response := sendTestBookingRead(router, test.path, test.token)
		if response.Code != 200 || !strings.Contains(response.Body.String(), `"status":"confirmed"`) {
			t.Fatalf("read confirmed=%d %s", response.Code, response.Body.String())
		}
		if test.token == "ADMIN" && !strings.Contains(response.Body.String(), "PAYMENT_CONFIRMED") {
			t.Fatal("staff history did not expose confirmation")
		}
	}
}
