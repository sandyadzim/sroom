package router

import (
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
	"payment/internal/repository"
	"payment/internal/service"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestExpiryAssessmentValidation(t *testing.T) {
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterConfirmation(app, handler.NewConfirmationHandler(nil, "test-only-pair-token"))
	path := "/internal/v1/bookings/00000000-0000-4000-8000-000000000001/expiry-assessment"
	for _, test := range []struct {
		token, path, body string
		status            int
	}{
		{"", path, "", 401}, {"customer-session", path, "", 401},
		{"test-only-pair-token", "/internal/v1/bookings/bad/expiry-assessment", "", 400},
		{"test-only-pair-token", path + "?status=expired", "", 400},
		{"test-only-pair-token", path, "{}", 400},
		{"test-only-pair-token", path, strings.Repeat("x", 16*1024+1), 413},
	} {
		req := httptest.NewRequest("POST", test.path, strings.NewReader(test.body))
		req.Header.Set("Authorization", "Bearer "+test.token)
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if res.Code != test.status {
			t.Fatalf("status=%d want=%d", res.Code, test.status)
		}
	}
}

func TestPrivateExpiryAssessmentPostgres(t *testing.T) {
	dsn := os.Getenv("PAYMENT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYMENT_TEST_DATABASE_URL to a dedicated migrated test database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal("could not open test database")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	bookingID, earlyID := uuid.NewString(), uuid.NewString()
	defer func() {
		if err := db.Exec("DELETE FROM payment.initiation_gates WHERE booking_id IN (?, ?)", bookingID, earlyID).Error; err != nil {
			t.Error(err)
		}
	}()
	deadline := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	bookingAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-only-pair-token" || r.Header.Get("X-Request-ID") != "expiry-http-test" {
			t.Error("missing pair auth or correlation")
		}
		id, holdDeadline := bookingID, deadline
		if strings.Contains(r.URL.Path, earlyID) {
			id, holdDeadline = earlyID, time.Now().Add(time.Hour)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": booking.PaymentContext{BookingID: id, CustomerID: "00000000-0000-4000-8000-000000000002", TotalIDR: 150000, Currency: "IDR", Status: "pending_payment", HoldExpiresAt: holdDeadline}})
	}))
	defer bookingAPI.Close()
	client, err := booking.NewClient(bookingAPI.URL, "test-only-pair-token")
	if err != nil {
		t.Fatal(err)
	}
	app, err := New(handler.NewHealthHandler(nil))
	if err != nil {
		t.Fatal(err)
	}
	RegisterConfirmation(app, handler.NewConfirmationHandler(service.NewPaymentService(repository.NewPaymentRepository(db), client, nil, false), "test-only-pair-token"))
	var closedAt *time.Time
	for _, id := range []string{bookingID, bookingID, earlyID} {
		req := httptest.NewRequest("POST", "/internal/v1/bookings/"+id+"/expiry-assessment", nil)
		req.Header.Set("Authorization", "Bearer test-only-pair-token")
		req.Header.Set("X-Request-ID", "expiry-http-test")
		res := httptest.NewRecorder()
		app.ServeHTTP(res, req)
		if id == earlyID {
			if res.Code != 409 || !strings.Contains(res.Body.String(), "BOOKING_NOT_EXPIRABLE") {
				t.Fatal("early closure accepted")
			}
			continue
		}
		var response struct {
			Data dto.ExpiryAssessment `json:"data"`
		}
		if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &response) != nil {
			t.Fatalf("expiry response status=%d", res.Code)
		}
		if response.Data.Result != "no_attempt_closed" || response.Data.ClosedAt == nil || response.Data.PaymentID != nil || !response.Data.HoldExpiresAt.Equal(deadline) || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("invalid closure response")
		}
		if closedAt != nil && !closedAt.Equal(*response.Data.ClosedAt) {
			t.Fatal("retry changed closure")
		}
		closedAt = response.Data.ClosedAt
	}
}
