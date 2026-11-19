package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking/internal/catalog"
	"booking/internal/checkout"
	"booking/internal/identity"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	quotePath      = "/api/v1/booking/checkout/quotes"
	quoteVariantID = "c67ee02e-85db-451f-9550-64e7edb8be7e"
	quoteRequestID = "quote-test"
)

func TestQuoteHTTP(t *testing.T) {
	for _, test := range []struct {
		name, token, pickupAt, returnAt, code string
		catalogStatus, status                 int
	}{
		{"guest", "", pickup, returnTime, "UNAUTHENTICATED", 200, 401},
		{"invalid session", "invalid", pickup, returnTime, "UNAUTHENTICATED", 200, 401},
		{"admin", "ADMIN", pickup, returnTime, "CUSTOMER_REQUIRED", 200, 403},
		{"identity down", "identity-down", pickup, returnTime, "DEPENDENCY_UNAVAILABLE", 200, 503},
		{"timestamp missing offset", "CUSTOMER", "2026-11-10T08:00:00", returnTime, "INVALID_REQUEST", 200, 400},
		{"invalid schedule", "CUSTOMER", returnTime, pickup, "INVALID_SCHEDULE", 200, 422},
		{"missing catalog token", "CUSTOMER", pickup, returnTime, "DEPENDENCY_UNAVAILABLE", 0, 503},
		{"catalog down", "CUSTOMER", pickup, returnTime, "DEPENDENCY_UNAVAILABLE", 500, 503},
		{"variant missing", "CUSTOMER", pickup, returnTime, "CAR_NOT_FOUND", 404, 404},
		{"inactive with no units", "CUSTOMER", pickup, returnTime, "", 200, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var client *catalog.Client
			if test.catalogStatus != 0 {
				client = quoteTestCatalog(t, test.catalogStatus, "inactive", []string{}, false)
			}
			router := quoteTestRouter(t, client, nil)
			res := sendTestQuote(router, test.token, test.pickupAt, test.returnAt)
			if res.Code != test.status || res.Header().Get("X-Request-ID") != quoteRequestID ||
				res.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("unexpected quote response: %d %s", res.Code, res.Body.String())
			}
			var response struct {
				Data  checkout.Quote
				Error struct{ Code string }
			}
			if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != test.code {
				t.Fatalf("code = %q, want %q", response.Error.Code, test.code)
			}
			if test.status == 200 && (response.Data.Available || response.Data.TotalIDR != 900000) {
				t.Fatalf("unexpected inactive quote: %+v", response.Data)
			}
			if strings.Contains(res.Body.String(), "synthetic private detail") {
				t.Fatal("dependency details leaked into quote response")
			}
		})
	}
}

func TestQuoteAvailabilityPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	customerID := newTestID(t, ctx, db)
	firstUnit := newTestID(t, ctx, db)
	secondUnit := newTestID(t, ctx, db)
	cleanupTestBookings(t, db, customerID)

	blockedBooking, err := bookTestUnit(ctx, db, customerID, firstUnit, pickup, returnTime)
	if err != nil {
		t.Fatal(err)
	}
	assertQuote := func(unitIDs []string, pickupAt, returnAt string, available bool, rentalDays int) {
		t.Helper()
		router := quoteTestRouter(t, quoteTestCatalog(t, 200, "active", unitIDs, false), db)
		res := sendTestQuote(router, "CUSTOMER", pickupAt, returnAt)
		var response struct{ Data checkout.Quote }
		if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil || res.Code != 200 {
			t.Fatalf("unexpected response: %d %s (%v)", res.Code, res.Body.String(), err)
		}
		if response.Data.Available != available || response.Data.RentalDays != rentalDays ||
			response.Data.TotalIDR != 450000*int64(rentalDays) || response.Data.Currency != "IDR" {
			t.Fatalf("unexpected quote: %+v, want available=%t and %d days", response.Data, available, rentalDays)
		}
		if strings.Contains(res.Body.String(), firstUnit) || strings.Contains(res.Body.String(), secondUnit) {
			t.Fatal("quote exposed the internal unit IDs")
		}
	}

	assertQuote([]string{}, pickup, nextReturn, false, 3)
	assertQuote([]string{secondUnit}, pickup, nextReturn, true, 3)
	assertQuote([]string{firstUnit, secondUnit}, pickup, nextReturn, true, 3)
	assertQuote([]string{firstUnit}, pickup, nextReturn, false, 3)
	assertQuote([]string{firstUnit}, returnTime, nextReturn, true, 2)
	if _, err := db.ExecContext(ctx, "UPDATE booking.allocations SET blocks_inventory = FALSE, released_at = NOW() WHERE booking_id = $1", blockedBooking); err != nil {
		t.Fatal(err)
	}
	assertQuote([]string{firstUnit}, pickup, nextReturn, true, 3)

	var bookings, allocations int
	err = db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM booking.bookings WHERE customer_id = $1),
		       (SELECT COUNT(*) FROM booking.allocations WHERE unit_id IN ($2, $3))
	`, customerID, firstUnit, secondUnit).Scan(&bookings, &allocations)
	if err != nil || bookings != 1 || allocations != 1 {
		t.Fatalf("expected only fixtures, got %d bookings and %d allocations (%v)", bookings, allocations, err)
	}
}

func quoteTestRouter(t *testing.T, client *catalog.Client, pool *sql.DB) *gin.Engine {
	t.Helper()
	identityServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/identity/me" || r.Header.Get("X-Request-ID") != quoteRequestID {
			t.Error("unexpected Identity route or request ID")
		}
		switch token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); token {
		case "CUSTOMER", "ADMIN", "OWNER":
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"id": testCustomerID, "role": token}})
		case "OTHER_CUSTOMER":
			_ = json.NewEncoder(w).Encode(gin.H{"data": gin.H{"id": otherBookingCustomer, "role": "CUSTOMER"}})
		case "identity-down":
			http.Error(w, "synthetic private detail", 500)
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	t.Cleanup(identityServer.Close)
	identityClient, err := identity.NewClient(identityServer.URL)
	if err != nil {
		t.Fatal(err)
	}

	var repository *checkout.Repository
	if pool != nil {
		db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true})
		if err != nil {
			t.Fatal(err)
		}
		repository = checkout.NewRepository(db)
	}
	checkoutService, err := checkout.NewService(client, repository)
	if err != nil {
		t.Fatal(err)
	}
	router, err := newRouter(func(context.Context) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time { return time.Date(2026, 11, 9, 15, 59, 0, 0, time.UTC) }
	registerCheckout(router, quoteHandler{identity: identityClient, checkout: checkoutService, now: now})
	return router
}

func quoteTestCatalog(t *testing.T, responseStatus int, status string, unitIDs []string, demoHolds bool) *catalog.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/variants/"+quoteVariantID+"/booking-snapshot" ||
			r.Header.Get("Authorization") != "Bearer fixture-token" || r.Header.Get("X-Request-ID") != quoteRequestID {
			t.Error("unexpected Catalog request")
		}
		w.WriteHeader(responseStatus)
		if responseStatus == 404 {
			_, _ = io.WriteString(w, `{"error":{"code":"CAR_NOT_FOUND"}}`)
			return
		}
		_ = json.NewEncoder(w).Encode(gin.H{"data": catalog.BookingSnapshot{
			VariantID: quoteVariantID, VehicleLabel: "Toyota Innova V", DailyRateIDR: 450000,
			Status: status, EligibleUnitIDs: unitIDs, DemoHoldsEnabled: demoHolds,
		}})
	}))
	t.Cleanup(server.Close)
	client, err := catalog.NewClient(server.URL, "fixture-token")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func sendTestQuote(router http.Handler, token, pickupAt, returnAt string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(quoteRequest{VariantID: quoteVariantID, PickupAt: pickupAt, ReturnAt: returnAt})
	req := httptest.NewRequest(http.MethodPost, quotePath, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", quoteRequestID)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}
