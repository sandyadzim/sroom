package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
)

const reportPath = "/api/v1/booking/admin/reports"
const overviewPath = "/api/v1/booking/admin/overview"

func TestReportAccessAndValidation(t *testing.T) {
	router := quoteTestRouter(t, nil, nil)
	for _, test := range []struct {
		path, token, code string
		status            int
	}{
		{reportPath, "", "UNAUTHENTICATED", 401}, {overviewPath, "", "UNAUTHENTICATED", 401},
		{reportPath, "CUSTOMER", "FORBIDDEN", 403}, {overviewPath, "CUSTOMER", "FORBIDDEN", 403},
		{reportPath, "expired-token", "UNAUTHENTICATED", 401}, {reportPath, "identity-down", "DEPENDENCY_UNAVAILABLE", 503},
		{overviewPath + "?from=2026-11-01", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?from=2026-11-01", "OWNER", "INVALID_REQUEST", 400},
		{reportPath + "?from=&to=", "OWNER", "INVALID_REQUEST", 400},
		{reportPath + "?from=2026-11-01&to=2026-11-02&sort=asc", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?from=2026-11-01&from=2026-11-02&to=2026-11-03", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?from=2026-02-30&to=2026-03-01", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?from=2026-11-03&to=2026-11-02", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?from=2024-01-01&to=2025-01-01", "ADMIN", "INVALID_REQUEST", 400},
		{reportPath + "?bad=%", "ADMIN", "INVALID_REQUEST", 400},
	} {
		res := sendTestBookingRead(router, test.path, test.token)
		if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) {
			t.Errorf("%s: %d %s", test.path, res.Code, res.Body.String())
		}
	}
}

// These are synthetic aggregation fixtures in a dedicated PostgreSQL database,
// not evidence of a payment or a production booking transition.
func TestReportsPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	customer := uuid.NewString()
	cleanupTestBookings(t, db, customer)
	variantA, variantB := uuid.NewString(), uuid.NewString()
	var ids []string
	fixtures := []struct {
		created, status, variant, label string
		verified                        bool
	}{
		{"2026-11-02T15:59:59Z", "confirmed", variantA, "Before range", true},
		{"2026-11-02T16:00:00Z", "pending_payment", variantA, "Old label", false},
		{"2026-11-03T01:00:00Z", "confirmed", variantA, "Saved price", true},
		{"2026-11-03T02:00:00Z", "confirmed", variantA, "Demo only", false},
		{"2026-11-03T03:00:00Z", "expired", variantB, "Other variant", false},
		{"2026-11-03T03:00:00Z", "payment_review", variantB, "Other variant latest", true},
		{"2026-11-09T15:59:59Z", "pending_payment", variantA, "Latest snapshot", false},
		{"2026-11-09T16:00:00Z", "confirmed", variantA, "After range", true},
	}
	for _, f := range fixtures {
		id, err := bookTestUnit(ctx, db, customer, uuid.NewString(), pickup, returnTime)
		if err != nil {
			t.Fatal(err)
		}
		var paymentID, orderID any
		if f.verified {
			paymentID = uuid.NewString()
			orderID = "synthetic-" + id
		}
		_, err = db.ExecContext(ctx, `UPDATE booking.bookings SET created_at=$2,status=$3,variant_id=$4,vehicle_label_snapshot=$5,confirmed_payment_id=$6,confirmed_provider_order_id=$7 WHERE id=$1`, id, f.created, f.status, f.variant, f.label, paymentID, orderID)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	router := quoteTestRouter(t, nil, db)
	res := sendTestBookingRead(router, reportPath+"?from=2026-11-03&to=2026-11-09", "ADMIN")
	if res.Code != 200 {
		t.Fatalf("reports: %d %s", res.Code, res.Body.String())
	}
	var reports struct{ Data checkout.Reports }
	if err := json.Unmarshal(res.Body.Bytes(), &reports); err != nil {
		t.Fatal(err)
	}
	data := reports.Data
	if data.Summary.TotalBookings != 6 || data.Summary.PendingPaymentBookings != 2 || data.Summary.ConfirmedBookings != 2 || data.Summary.ExpiredBookings != 1 || data.Summary.PaymentReviewBookings != 1 || data.Summary.VerifiedConfirmedBookings != 1 || data.Summary.ConfirmedValueIDR != 800000 || data.Summary.PaymentConversion == nil || *data.Summary.PaymentConversion != 1.0/6.0 {
		t.Fatalf("wrong aggregate: %+v", data.Summary)
	}
	if len(data.Daily) != 7 || data.Daily[0].Bookings != 5 || data.Daily[0].ConfirmedValueIDR != 800000 || data.Daily[1].Bookings != 0 || data.Daily[6].Bookings != 1 || len(data.TopVariants) != 2 || data.TopVariants[0].VariantID != variantA || data.TopVariants[0].Bookings != 4 || data.TopVariants[0].VehicleLabel != "Latest snapshot" {
		t.Fatalf("wrong daily/ranking: %+v", data)
	}
	tied := []string{ids[4], ids[5]}
	slices.Sort(tied)
	latestLabel := "Other variant"
	if tied[1] == ids[5] {
		latestLabel = "Other variant latest"
	}
	if data.TopVariants[1].VehicleLabel != latestLabel {
		t.Fatalf("snapshot label tie not stable: %+v", data.TopVariants[1])
	}
	// The read path depends only on Booking snapshots, even without a Catalog client.
	res = sendTestBookingRead(router, overviewPath, "OWNER")
	var overview struct{ Data checkout.Overview }
	if err := json.Unmarshal(res.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || overview.Data.Period.From != "2026-11-03" || len(overview.Data.RecentBookings) != 5 || overview.Data.RecentBookings[0].ID != ids[6] || overview.Data.RecentBookings[4].ID != ids[2] {
		t.Fatalf("wrong overview: %d %s", res.Code, res.Body.String())
	}
	if overview.Data.RecentBookings[1].ID != tied[1] || overview.Data.RecentBookings[2].ID != tied[0] {
		t.Fatal("recent booking ties not ordered by ID descending")
	}
	res = sendTestBookingRead(router, reportPath+"?from=2040-01-01&to=2040-01-02", "OWNER")
	if err := json.Unmarshal(res.Body.Bytes(), &reports); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || reports.Data.Summary.TotalBookings != 0 || reports.Data.Summary.PaymentConversion != nil || reports.Data.TopVariants == nil || len(reports.Data.TopVariants) != 0 || len(reports.Data.Daily) != 2 {
		t.Fatalf("wrong empty report: %s", res.Body.String())
	}
}

func TestReportDatabaseUnavailable(t *testing.T) {
	db := openAllocationTestDB(t)
	router := quoteTestRouter(t, nil, db)
	// A failed database read must not become a successful report containing zeros.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	res := sendTestBookingRead(router, overviewPath, "ADMIN")
	if res.Code != 503 || !strings.Contains(res.Body.String(), `"DEPENDENCY_UNAVAILABLE"`) {
		t.Fatalf("database failure: %d %s", res.Code, res.Body.String())
	}
}

func TestReportRankingLimitPostgres(t *testing.T) {
	db := openAllocationTestDB(t)
	customer := uuid.NewString()
	cleanupTestBookings(t, db, customer)
	ctx := t.Context()
	variants := []string{}
	for range 12 {
		variant := uuid.NewString()
		variants = append(variants, variant)
		id, err := bookTestUnit(ctx, db, customer, uuid.NewString(), pickup, returnTime)
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(ctx, `UPDATE booking.bookings SET created_at='2041-01-01T00:00:00+08:00',variant_id=$2 WHERE id=$1`, id, variant)
		if err != nil {
			t.Fatal(err)
		}
	}
	slices.Sort(variants)
	router := quoteTestRouter(t, nil, db)
	res := sendTestBookingRead(router, reportPath+"?from=2041-01-01&to=2041-01-01", "OWNER")
	var result struct{ Data checkout.Reports }
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || result.Data.Summary.TotalBookings != 12 || len(result.Data.TopVariants) != 10 {
		t.Fatalf("wrong ranking: %d %s", res.Code, res.Body.String())
	}
	for i, row := range result.Data.TopVariants {
		if row.VariantID != variants[i] || row.Bookings != 1 {
			t.Fatal("ranking tie must use variant ID ascending")
		}
	}
}
