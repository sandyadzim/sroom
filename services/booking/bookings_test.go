package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
)

const (
	otherBookingCustomer = "c67ee02e-85db-451f-9550-64e7edb8be7e"
	customerBookingsPath = "/api/v1/booking/bookings"
	staffBookingsPath    = "/api/v1/booking/admin/bookings"
)

func TestBookingsReadAccessAndValidation(t *testing.T) {
	router := quoteTestRouter(t, nil, nil)
	for _, test := range []struct {
		path, token, code string
		status            int
	}{
		{customerBookingsPath, "", "UNAUTHENTICATED", 401},
		{staffBookingsPath, "", "UNAUTHENTICATED", 401},
		{staffBookingsPath + "/" + uuid.NewString(), "CUSTOMER", "FORBIDDEN", 403},
		{staffBookingsPath, "CUSTOMER", "FORBIDDEN", 403},
		{customerBookingsPath, "ADMIN", "CUSTOMER_REQUIRED", 403},
		{customerBookingsPath + "/invalid", "CUSTOMER", "INVALID_REQUEST", 400},
		{customerBookingsPath + "?status=paid", "CUSTOMER", "INVALID_REQUEST", 400},
		{customerBookingsPath + "?limit=101", "CUSTOMER", "INVALID_REQUEST", 400},
		{customerBookingsPath + "?search=fixture", "CUSTOMER", "INVALID_REQUEST", 400},
		{staffBookingsPath + "?page=0", "OWNER", "INVALID_REQUEST", 400},
		{staffBookingsPath + "?page=1&page=2", "OWNER", "INVALID_REQUEST", 400},
		{staffBookingsPath + "?pickup_date=2026-02-30", "OWNER", "INVALID_REQUEST", 400},
		{staffBookingsPath + "?variant_id=invalid", "OWNER", "INVALID_REQUEST", 400},
	} {
		res := sendTestBookingRead(router, test.path, test.token)
		if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) {
			t.Errorf("%s as %s: got %d %s, want %d %s", test.path, test.token, res.Code, res.Body.String(), test.status, test.code)
		}
	}
}

func TestReadBookingsPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cleanupTestBookings(t, db, testCustomerID)
	cleanupTestBookings(t, db, otherBookingCustomer)

	book := func(customer, unit, pickupAt, returnAt string) string {
		t.Helper()
		id, err := bookTestUnit(ctx, db, customer, unit, pickupAt, returnAt)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	unitID := uuid.NewString()
	first := book(testCustomerID, unitID, pickup, returnTime)
	second := book(testCustomerID, unitID, returnTime, nextReturn)
	other := book(otherBookingCustomer, uuid.NewString(), pickup, returnTime)
	variantID := uuid.NewString()
	_, err := db.ExecContext(ctx, `
		UPDATE booking.bookings SET vehicle_label_snapshot = 'Snapshot INNOVA', variant_id = $2
		WHERE id = $1
	`, first, variantID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
		UPDATE booking.bookings SET created_at = '2026-11-09T12:00:00Z' WHERE id IN ($1, $2)
	`, first, second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE booking.bookings SET status = 'expired' WHERE id = $1", second); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := db.ExecContext(ctx, "DELETE FROM booking.booking_status_history WHERE booking_id = $1", first); err != nil {
			t.Error(err)
		}
	})
	_, err = db.ExecContext(ctx, `
		INSERT INTO booking.booking_status_history (booking_id, from_status, to_status, actor_type, occurred_at)
		VALUES ($1, 'pending_payment', 'payment_review', 'system', '2026-11-09T15:05:00Z'),
		       ($1, NULL, 'pending_payment', 'customer', '2026-11-09T15:00:00Z')
	`, first)
	if err != nil {
		t.Fatal(err)
	}

	router := quoteTestRouter(t, nil, db)
	read := func(path, token string, status int, target any) []byte {
		t.Helper()
		res := sendTestBookingRead(router, path, token)
		if res.Code != status || res.Header().Get("Cache-Control") != "no-store" ||
			res.Header().Get("X-Request-ID") != quoteRequestID {
			t.Fatalf("%s returned %d %s, want %d", path, res.Code, res.Body.String(), status)
		}
		if target != nil {
			if err := json.Unmarshal(res.Body.Bytes(), target); err != nil {
				t.Fatal(err)
			}
		}
		return res.Body.Bytes()
	}

	wantOrder := []string{first, second}
	slices.Sort(wantOrder)
	slices.Reverse(wantOrder)
	for i, page := range []string{"1", "2"} {
		var result struct {
			Data []checkout.Booking
			Page checkout.BookingPage
		}
		read(customerBookingsPath+"?limit=1&page="+page, "CUSTOMER", 200, &result)
		if len(result.Data) != 1 || result.Data[0].ID != wantOrder[i] ||
			result.Page != (checkout.BookingPage{Number: i + 1, Limit: 1, Total: 2}) {
			t.Fatalf("unexpected customer page: %+v", result)
		}
	}
	var customerPage struct {
		Data []checkout.Booking
		Page checkout.BookingPage
	}
	read(customerBookingsPath, "OTHER_CUSTOMER", 200, &customerPage)
	if len(customerPage.Data) != 1 || customerPage.Data[0].ID != other ||
		customerPage.Page != (checkout.BookingPage{Number: 1, Limit: 20, Total: 1}) {
		t.Fatalf("unexpected other customer page: %+v", customerPage)
	}
	read(customerBookingsPath+"?status=expired", "CUSTOMER", 200, &customerPage)
	if len(customerPage.Data) != 1 || customerPage.Data[0].ID != second || customerPage.Page.Total != 1 {
		t.Fatalf("unexpected status-filtered page: %+v", customerPage)
	}
	read(customerBookingsPath+"?page=2", "CUSTOMER", 200, &customerPage)
	if customerPage.Data == nil || len(customerPage.Data) != 0 || customerPage.Page.Total != 2 {
		t.Fatalf("unexpected empty page: %+v", customerPage)
	}

	missing := read(customerBookingsPath+"/"+uuid.NewString(), "CUSTOMER", 404, nil)
	for _, inaccessible := range []struct{ id, token string }{{other, "CUSTOMER"}, {first, "OTHER_CUSTOMER"}} {
		if body := read(customerBookingsPath+"/"+inaccessible.id, inaccessible.token, 404, nil); string(body) != string(missing) {
			t.Fatal("customer detail reveals whether another customer's booking exists")
		}
	}
	var detail struct{ Data checkout.Booking }
	body := read(customerBookingsPath+"/"+first, "CUSTOMER", 200, &detail)
	if detail.Data.ID != first || detail.Data.TotalIDR != 800000 || detail.Data.DailyRateIDR != 400000 ||
		detail.Data.VehicleLabel != "Snapshot INNOVA" || detail.Data.Status != "pending_payment" ||
		!strings.Contains(string(body), "+08:00") {
		t.Fatalf("unexpected booking snapshot: %+v", detail.Data)
	}
	for _, field := range []string{`"unit_id"`, `"customer_id"`, `"status_history"`, `"payment_status"`} {
		if strings.Contains(string(body), field) {
			t.Errorf("customer response exposed %s", field)
		}
	}

	for _, search := range []string{"innova", strings.ToUpper(first[:8]), strings.ToUpper(testCustomerID[:8])} {
		var result struct {
			Data []checkout.StaffBooking
			Page checkout.BookingPage
		}
		path := staffBookingsPath + "?pickup_date=2026-11-10&status=pending_payment&variant_id=" + variantID + "&search=" + search
		read(path, "ADMIN", 200, &result)
		if result.Page.Total != 1 || len(result.Data) != 1 || result.Data[0].ID != first ||
			result.Data[0].CustomerID != testCustomerID || result.Data[0].UnitID == nil || *result.Data[0].UnitID != unitID {
			t.Fatalf("unexpected staff search results: %+v", result)
		}
	}
	var staff struct{ Data checkout.StaffBookingDetail }
	read(staffBookingsPath+"/"+first, "OWNER", 200, &staff)
	if len(staff.Data.StatusHistory) != 2 || staff.Data.StatusHistory[0].FromStatus != nil ||
		staff.Data.StatusHistory[0].ToStatus != "pending_payment" || staff.Data.StatusHistory[1].ToStatus != "payment_review" {
		t.Fatalf("unexpected status history: %+v", staff.Data.StatusHistory)
	}
	read(staffBookingsPath+"/"+other, "ADMIN", 200, nil)
}

func sendTestBookingRead(router http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("X-Request-ID", quoteRequestID)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}
