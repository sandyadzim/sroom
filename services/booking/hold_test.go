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

	"github.com/google/uuid"
)

const testCustomerID = "4eee8620-3773-45ca-bc88-528dfc7712d2"

func TestCreateDemoHoldPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	first, second := uuid.NewString(), uuid.NewString()
	cleanupTestBookings(t, db, testCustomerID)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, statement := range []string{
			"DELETE FROM booking.outbox_events WHERE aggregate_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.booking_status_history WHERE booking_id IN (SELECT id FROM booking.bookings WHERE customer_id = $1)",
			"DELETE FROM booking.idempotency_records WHERE customer_id = $1",
		} {
			if _, err := db.ExecContext(ctx, statement, testCustomerID); err != nil {
				t.Error(err)
			}
		}
	})

	router := quoteTestRouter(t, quoteTestCatalog(t, 200, "active", []string{first, second}, true), db)
	disabled := quoteTestRouter(t, quoteTestCatalog(t, 200, "active", []string{first, second}, false), db)
	payload := holdRequest{
		quoteRequest:     quoteRequest{VariantID: quoteVariantID, PickupAt: pickup, ReturnAt: nextReturn},
		ExpectedTotalIDR: 1350000, TermsVersion: "rental-terms-v1", AcceptTerms: true,
	}
	send := func(router http.Handler, key string, payload holdRequest, status int, code string) holdTestResponse {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/booking/bookings", strings.NewReader(string(body)))
		req.Header.Set("Authorization", "Bearer CUSTOMER")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", quoteRequestID)
		req.Header.Set("Idempotency-Key", key)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		var response holdTestResponse
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || res.Code != status || response.Error.Code != code {
			t.Fatalf("response = %d %s, want %d %q (%v)", res.Code, res.Body.String(), status, code, err)
		}
		return response
	}

	send(disabled, uuid.NewString(), payload, 503, "HOLD_NOT_ENABLED")
	noTerms := payload
	noTerms.AcceptTerms = false
	send(router, uuid.NewString(), noTerms, 422, "INVALID_TERMS")
	priceChanged := payload
	priceChanged.ExpectedTotalIDR = 1000000
	revised := send(router, uuid.NewString(), priceChanged, 409, "QUOTE_CHANGED").Error.Details.Quote
	if revised.TotalIDR != 1350000 || !revised.Available {
		t.Fatalf("unexpected revised quote: %+v", revised)
	}

	key := uuid.NewString()
	firstHold := send(router, key, payload, 201, "").Data
	replayed := send(router, key, payload, 201, "").Data
	if firstHold.ID == "" || firstHold.Status != "pending_payment" || firstHold.PaymentStatus != "not_started" ||
		firstHold.TotalIDR != 1350000 || !firstHold.HoldExpiresAt.Equal(time.Date(2026, 11, 9, 16, 14, 0, 0, time.UTC)) ||
		replayed != firstHold {
		t.Fatalf("unexpected hold or replay: %+v, %+v", firstHold, replayed)
	}
	send(router, key, priceChanged, 409, "IDEMPOTENCY_CONFLICT")

	secondHold := send(router, uuid.NewString(), payload, 201, "").Data
	send(router, uuid.NewString(), payload, 409, "AVAILABILITY_CONFLICT")
	for _, result := range []struct{ id, unitID string }{{firstHold.ID, first}, {secondHold.ID, second}} {
		var unitID, eventType, eventCustomer, terms, actor string
		var storedTotal int64
		var storedPickup, storedReturn time.Time
		err := db.QueryRowContext(ctx, `
			SELECT a.unit_id, o.event_type, o.envelope->'payload'->>'customer_id', b.terms_version,
			       h.actor_type, b.total_idr, b.pickup_at, b.return_at
			FROM booking.bookings b
			JOIN booking.allocations a ON a.booking_id = b.id
			JOIN booking.outbox_events o ON o.aggregate_id = b.id
			JOIN booking.booking_status_history h ON h.booking_id = b.id
			WHERE b.id = $1
		`, result.id).Scan(&unitID, &eventType, &eventCustomer, &terms, &actor, &storedTotal, &storedPickup, &storedReturn)
		if err != nil || unitID != result.unitID || eventType != "booking.created.v1" || eventCustomer != testCustomerID ||
			terms != "rental-terms-v1" || actor != "customer" || storedTotal != firstHold.TotalIDR ||
			!storedPickup.Equal(firstHold.PickupAt) || !storedReturn.Equal(firstHold.ReturnAt) {
			t.Fatalf("booking, history, allocation or event was not saved as expected: %v", err)
		}
	}
	var bookings int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM booking.bookings WHERE customer_id = $1", testCustomerID).Scan(&bookings); err != nil || bookings != 2 {
		t.Fatalf("found %d bookings, want two with no orphan bookings (%v)", bookings, err)
	}
	confirmationRouter := demoConfirmationTestRouter(t, db, paymentTestToken, true, firstHold.HoldExpiresAt.Add(-time.Minute))
	confirmation := demoConfirmationRequestForTest(confirmationRouter, firstHold.ID, paymentTestToken, `{"amount_idr":1350000,"currency":"IDR"}`)
	if confirmation.Code != 200 {
		t.Fatalf("confirming the created hold: %d %s", confirmation.Code, confirmation.Body.String())
	}
	for _, token := range []string{"CUSTOMER", "ADMIN"} {
		path := "/api/v1/booking/bookings/" + firstHold.ID
		if token == "ADMIN" {
			path = "/api/v1/booking/admin/bookings/" + firstHold.ID
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Request-ID", quoteRequestID)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		var response struct{ Data checkout.StaffBookingDetail }
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || res.Code != 200 || response.Data.Status != "confirmed" || response.Data.TotalIDR != 1350000 {
			t.Fatalf("reading the confirmed hold: %d %s (%v)", res.Code, res.Body.String(), err)
		}
		if token == "ADMIN" && (len(response.Data.StatusHistory) != 2 || response.Data.StatusHistory[1].ReasonCode == nil || *response.Data.StatusHistory[1].ReasonCode != "DEMO_PAYMENT_CONFIRMED") {
			t.Fatalf("unexpected staff history: %+v", response.Data.StatusHistory)
		}
	}
}

type holdTestResponse struct {
	Data  checkout.Hold
	Error struct {
		Code    string
		Details struct{ Quote checkout.Quote }
	}
}
