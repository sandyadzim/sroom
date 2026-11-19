package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/handler"
	"payment/internal/identity"
	"payment/internal/midtrans"
	"payment/internal/repository"
	"payment/internal/service"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type integrationProvider struct{ calls int }

func (p *integrationProvider) CreateVA(_ context.Context, input midtrans.ChargeInput, _ string) (midtrans.ChargeResult, error) {
	p.calls++
	return midtrans.ChargeResult{OrderID: input.OrderID, TransactionID: input.OrderID, AmountIDR: input.AmountIDR,
		Currency: "IDR", Bank: "bni", Status: "pending", VANumber: "0000000000000000", ExpiresAt: input.HoldExpiresAt.Truncate(time.Second)}, nil
}

func TestPaymentHTTPFlowPostgres(t *testing.T) {
	dsn := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYMENT_TEST_DATABASE_URL to a dedicated, migrated PostgreSQL test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("could not open test database")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error("could not close HTTP test pool")
		}
	})
	customerA, customerB, bookingID, key := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, table := range []string{"provider_observations", "idempotency_records"} {
			if err := db.WithContext(ctx).Exec("DELETE FROM payment."+table+" WHERE payment_id IN (SELECT id FROM payment.payment_attempts WHERE customer_id = ?)", customerA).Error; err != nil {
				t.Error("could not clean HTTP test rows")
			}
		}
		if err := db.WithContext(ctx).Exec("DELETE FROM payment.payment_attempts WHERE customer_id = ?", customerA).Error; err != nil {
			t.Error("could not clean HTTP test attempt")
		}
	})
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/me" || r.Header.Get("X-Request-ID") != "http-va-flow" {
			t.Error("incorrect Identity request")
		}
		id := customerA
		if r.Header.Get("Authorization") == "Bearer customer-b" {
			id = customerB
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"id": id, "role": "CUSTOMER"}})
	}))
	defer identityServer.Close()
	deadline := time.Now().UTC().Add(15 * time.Minute).Truncate(time.Second)
	bookingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-only-pair-token" || r.Header.Get("X-Request-ID") != "http-va-flow" {
			t.Error("incorrect Booking credentials/correlation")
		}
		if r.URL.Path != "/internal/v1/bookings/"+bookingID+"/payment-context" {
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": booking.PaymentContext{BookingID: bookingID, CustomerID: customerA,
			TotalIDR: 150000, Currency: "IDR", Status: "pending_payment", HoldExpiresAt: deadline}})
	}))
	defer bookingServer.Close()
	identityClient, err := identity.NewClient(identityServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	bookingClient, err := booking.NewClient(bookingServer.URL, "test-only-pair-token")
	if err != nil {
		t.Fatal(err)
	}
	provider := &integrationProvider{}
	engine, err := New(handler.NewHealthHandler(pool.PingContext))
	if err != nil {
		t.Fatal(err)
	}
	RegisterPayment(engine, identityClient, handler.NewPaymentHandler(service.NewPaymentService(repository.NewPaymentRepository(db), bookingClient, provider, true)))
	var firstID string
	for _, test := range []struct {
		token, id string
		status    int
	}{{"customer-a", bookingID, 201}, {"customer-a", bookingID, 200}, {"customer-b", bookingID, 404}, {"customer-a", uuid.NewString(), 404}} {
		req := httptest.NewRequest("POST", "/api/v1/payment/bookings/"+test.id+"/payments/va", strings.NewReader("{}"))
		req.Header.Set("Authorization", "Bearer "+test.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("X-Request-ID", "http-va-flow")
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != test.status {
			t.Fatalf("got %d %s", res.Code, res.Body.String())
		}
		if res.Code == 201 || res.Code == 200 {
			var body struct {
				Data dto.VAResponse `json:"data"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Data.AmountIDR != 150000 || body.Data.VANumber == nil || body.Data.Status != "pending" {
				t.Fatal("incorrect VA response")
			}
			if firstID != "" && body.Data.PaymentID != firstID {
				t.Fatal("retry created a different attempt")
			}
			firstID = body.Data.PaymentID
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls=%d", provider.calls)
	}
}
