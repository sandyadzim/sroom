package router

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
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

func (p *integrationProvider) GetStatus(_ context.Context, orderID, _ string) (midtrans.StatusResult, error) {
	return midtrans.StatusResult{OrderID: orderID, TransactionID: orderID, MerchantID: "synthetic-merchant", AmountIDR: 150000,
		Currency: "IDR", Bank: "bni", PaymentType: "bank_transfer", Status: "settlement", StatusCode: "200", FraudStatus: "accept"}, nil
}

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
		if err := db.WithContext(ctx).Exec("DELETE FROM payment.outbox_events WHERE aggregate_id IN (SELECT id FROM payment.payment_attempts WHERE customer_id = ?)", customerA).Error; err != nil {
			t.Error("could not clean HTTP test outbox")
		}
		for _, table := range []string{"provider_observations", "webhook_receipts", "idempotency_records"} {
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
		role := "CUSTOMER"
		if r.Header.Get("Authorization") == "Bearer admin" {
			role = "ADMIN"
		}
		if r.Header.Get("Authorization") == "Bearer owner" {
			role = "OWNER"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]string{"id": id, "role": role}})
	}))
	defer identityServer.Close()
	deadline := time.Now().UTC().Add(15 * time.Minute).Truncate(time.Second)
	var bookingUnavailable atomic.Bool
	bookingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if bookingUnavailable.Load() {
			w.WriteHeader(503)
			return
		}
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
	paymentService := service.NewPaymentService(repository.NewPaymentRepository(db), bookingClient, provider, true)
	RegisterPayment(engine, identityClient, handler.NewPaymentHandler(paymentService))
	RegisterPaymentReads(engine, identityClient, handler.NewReadPaymentHandler(paymentService))
	read := func(path, token string, status int) string {
		t.Helper()
		req := httptest.NewRequest("GET", path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("X-Request-ID", "http-va-flow")
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != status || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("read %s=%d %s", path, res.Code, res.Body.String())
		}
		return res.Body.String()
	}
	customerPath := "/api/v1/payment/bookings/" + bookingID + "/payment"
	if body := read(customerPath, "customer-a", 200); !strings.Contains(body, `"status":"not_started"`) || !strings.Contains(body, `"payment_id":null`) {
		t.Fatal("missing not_started response")
	}
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
	if body := read(customerPath, "customer-a", 200); !strings.Contains(body, `"status":"pending"`) || !strings.Contains(body, `"va_number":"`) || strings.Contains(body, "review_reason") {
		t.Fatal("incorrect customer read")
	}
	read(customerPath, "customer-b", 404)
	read("/api/v1/payment/bookings/"+uuid.NewString()+"/payment", "customer-a", 404)
	staffPath := "/api/v1/payment/admin/payments"
	read(staffPath, "", 401)
	read(staffPath, "customer-a", 403)
	for _, token := range []string{"admin", "owner"} {
		body := read(staffPath+"?search="+firstID, token, 200)
		if !strings.Contains(body, `"total":1`) || strings.Contains(body, "va_number") {
			t.Fatal("incorrect staff list")
		}
		body = read(staffPath+"/"+firstID, token, 200)
		if !strings.Contains(body, `"booking_status":"pending_payment"`) || !strings.Contains(body, `"source":"charge"`) || strings.Contains(body, "va_number") || strings.Contains(body, "payload_digest") {
			t.Fatal("incorrect staff detail")
		}
	}
	read(staffPath+"/"+uuid.NewString(), "admin", 404)
	bookingUnavailable.Store(true)
	read(customerPath, "customer-a", 503)
	read(staffPath+"/"+firstID, "admin", 503)
	read(staffPath+"?search="+firstID, "admin", 200)
	bookingUnavailable.Store(false)
	const serverKey = "synthetic-webhook-key"
	RegisterWebhook(engine, handler.NewWebhookHandler(service.NewWebhookService(repository.NewPaymentRepository(db), provider, serverKey)))
	var orderID string
	if err := db.Raw("SELECT provider_order_id FROM payment.payment_attempts WHERE id = ?", firstID).Scan(&orderID).Error; err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum512([]byte(orderID + "200" + "150000.00" + serverKey))
	body, err := json.Marshal(map[string]any{"order_id": orderID, "status_code": "200", "gross_amount": "150000.00", "signature_key": hex.EncodeToString(digest[:]), "future_field": true})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		// Provider callback requires neither a customer token nor an Origin header.
		req := httptest.NewRequest("POST", "/webhooks/midtrans", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		if res.Code != 200 || res.Body.String() != `{"status":"ok"}` {
			t.Fatalf("webhook=%d %s", res.Code, res.Body.String())
		}
	}
	var status string
	if err := db.Raw("SELECT status FROM payment.payment_attempts WHERE id = ?", firstID).Scan(&status).Error; err != nil {
		t.Fatal(err)
	}
	if status != "paid" {
		t.Fatal("HTTP callback did not persist settlement")
	}
	if body := read(customerPath, "customer-a", 200); !strings.Contains(body, `"status":"paid"`) || !strings.Contains(body, `"booking_status":"pending_payment"`) || !strings.Contains(body, `"va_number":null`) {
		t.Fatal("paid was not separated from Booking confirmation")
	}
	// A separate closed pool exercises a database outage without disturbing cleanup.
	unavailableDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("could not open failure-test pool")
	}
	unavailablePool, err := unavailableDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := unavailablePool.Close(); err != nil {
		t.Fatal(err)
	}
	unavailableEngine, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterWebhook(unavailableEngine, handler.NewWebhookHandler(service.NewWebhookService(repository.NewPaymentRepository(unavailableDB), provider, serverKey)))
	RegisterPaymentReads(unavailableEngine, identityClient, handler.NewReadPaymentHandler(service.NewPaymentService(repository.NewPaymentRepository(unavailableDB), bookingClient, provider, false)))
	readRequest := httptest.NewRequest("GET", staffPath, nil)
	readRequest.Header.Set("Authorization", "Bearer admin")
	readRequest.Header.Set("X-Request-ID", "http-va-flow")
	readResponse := httptest.NewRecorder()
	unavailableEngine.ServeHTTP(readResponse, readRequest)
	if readResponse.Code != 503 || strings.Contains(readResponse.Body.String(), "database is closed") {
		t.Fatal("unsafe database read failure")
	}
	req := httptest.NewRequest("POST", "/webhooks/midtrans", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	unavailableEngine.ServeHTTP(res, req)
	if res.Code != 503 || strings.Contains(res.Body.String(), "database is closed") {
		t.Fatalf("unsafe database failure response=%d %s", res.Code, res.Body.String())
	}
}
