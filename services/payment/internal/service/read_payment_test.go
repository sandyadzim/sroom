package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/repository"

	"github.com/google/uuid"
)

func TestCustomerPaymentNotStartedAndOwnershipPostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	svc.configured = false // Reads do not require a Midtrans key.
	result, err := svc.CustomerPayment(t.Context(), input.BookingID, input.CustomerID, "not-started")
	if err != nil || result.Status != "not_started" || result.PaymentID != nil || result.Bank != nil || result.VANumber != nil || result.ReviewRequired || result.AmountIDR != 150000 || result.BookingStatus != "pending_payment" {
		t.Fatalf("not started=%+v error=%v", result, err)
	}
	if countPaymentRows(t, db, "payment_attempts", "customer_id", input.CustomerID) != 0 || provider.calls != 0 {
		t.Fatal("GET created a payment")
	}
	for _, test := range []struct {
		name       string
		err        error
		customerID string
		want       error
	}{
		{"other customer", nil, uuid.NewString(), booking.ErrNotFound},
		{"missing", booking.ErrNotFound, input.CustomerID, booking.ErrNotFound},
		{"booking down", booking.ErrUnavailable, input.CustomerID, booking.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader.err = test.err
			_, err := svc.CustomerPayment(t.Context(), input.BookingID, test.customerID, "ownership")
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
		})
	}
	reader.err = nil
	reader.snapshot.CustomerID = "invalid"
	if _, err := svc.CustomerPayment(t.Context(), input.BookingID, input.CustomerID, "invalid-booking"); !errors.Is(err, booking.ErrUnavailable) {
		t.Fatalf("malformed booking=%v", err)
	}
}

func TestCustomerPaymentStoredStatusAndInstructionsPostgres(t *testing.T) {
	for _, state := range []string{"pending", "hold deadline", "provider deadline", "review", "initiating", "paid", "expired", "failed"} {
		t.Run(state, func(t *testing.T) {
			svc, db, reader, provider, input := paymentFixture(t)
			payment, _, err := svc.CreateVA(t.Context(), input, "create")
			if err != nil {
				t.Fatal(err)
			}
			svc.configured = false
			reader.snapshot.TotalIDR = 900000 // A read must preserve the original Payment snapshot.
			wantStatus := state
			switch state {
			case "hold deadline":
				svc.now = func() time.Time { return payment.HoldExpiresAt }
				wantStatus = "pending"
			case "provider deadline":
				if err := db.Exec("UPDATE payment.payment_attempts SET provider_expires_at = ? WHERE id = ?", svc.now(), payment.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
				wantStatus = "pending"
			case "review":
				if err := db.Exec("UPDATE payment.payment_attempts SET review_reason = 'SYNTHETIC_REVIEW' WHERE id = ?", payment.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
				wantStatus = "pending"
			default:
				if err := db.Exec("UPDATE payment.payment_attempts SET status = ?, verified_at = ? WHERE id = ?", state, svc.now(), payment.PaymentID).Error; err != nil {
					t.Fatal(err)
				}
			}
			result, err := svc.CustomerPayment(t.Context(), input.BookingID, input.CustomerID, "read")
			if err != nil || result.Status != wantStatus || result.AmountIDR != 150000 || result.PaymentID == nil || *result.PaymentID != payment.PaymentID {
				t.Fatalf("read=%+v error=%v", result, err)
			}
			if (result.VANumber != nil) != (state == "pending") || result.ReviewRequired != (state == "review") {
				t.Fatal("VA disclosure rules failed")
			}
			if state == "paid" && (result.VerifiedAt == nil || result.ProviderPaidAt != nil || result.BookingStatus != "pending_payment") {
				t.Fatal("payment and Booking status were combined")
			}
			if provider.calls != 1 || countPaymentRows(t, db, "provider_observations", "payment_id", payment.PaymentID) != 1 || countPaymentRows(t, db, "outbox_events", "aggregate_id", payment.PaymentID) != 0 {
				t.Fatal("GET caused a provider call or write")
			}
			saved := reloadAttempt(t, db, payment.PaymentID)
			if saved.Status != wantStatus {
				t.Fatal("GET changed stored status")
			}
			if _, err := svc.CustomerPayment(t.Context(), input.BookingID, uuid.NewString(), "other-customer"); !errors.Is(err, booking.ErrNotFound) {
				t.Fatal("cross-customer payment exposed")
			}
			// Check ownership against both Booking and the saved attempt.
			reader.snapshot.CustomerID = uuid.NewString()
			if _, err := svc.CustomerPayment(t.Context(), input.BookingID, reader.snapshot.CustomerID, "mismatched-owner"); !errors.Is(err, booking.ErrNotFound) {
				t.Fatal("stored Payment owner ignored")
			}
		})
	}
}

func TestStaffListPaymentsPostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	prefix := "pay005-" + uuid.NewString()[:20]
	ids := make([]string, 0, 3)
	for i := range 3 {
		input.BookingID = uuid.NewString()
		input.IdempotencyKey = uuid.NewString()
		reader.snapshot.BookingID = input.BookingID
		payment, _, err := svc.CreateVA(t.Context(), input, "create")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, payment.PaymentID)
		status := "pending"
		var review any
		orderID := prefix + "-normal"
		if i == 1 {
			status = "failed"
			review = "SYNTHETIC_REVIEW"
			orderID = prefix + "-literal%_\\"
		}
		if i == 2 {
			orderID = prefix + "-third"
		}
		if err := db.Exec("UPDATE payment.payment_attempts SET status = ?, review_reason = ?, provider_order_id = ?, created_at = ? WHERE id = ?", status, review, orderID, svc.now(), payment.PaymentID).Error; err != nil {
			t.Fatal(err)
		}
	}
	// List is Payment-only and remains usable when Booking is unavailable.
	reader.err = booking.ErrUnavailable
	svc.configured = false
	query := dto.ListPaymentsQuery{Page: 1, Limit: 2, Search: strings.ToUpper(prefix)}
	first, total, err := svc.ListPayments(t.Context(), query)
	if err != nil || total != 3 || len(first) != 2 || first[0].PaymentID < first[1].PaymentID || first[0].Provider != "midtrans" {
		t.Fatalf("list total=%d len=%d error=%v", total, len(first), err)
	}
	query.Page = 2
	second, total, err := svc.ListPayments(t.Context(), query)
	if err != nil || total != 3 || len(second) != 1 || second[0].PaymentID >= first[1].PaymentID {
		t.Fatal("unstable pagination")
	}
	query.Page = 3
	empty, total, err := svc.ListPayments(t.Context(), query)
	if err != nil || total != 3 || empty == nil || len(empty) != 0 {
		t.Fatal("empty pagination shape incorrect")
	}
	for _, test := range []struct {
		status, review, search string
		total                  int64
	}{
		{"pending", "", prefix, 2}, {"failed", "required", prefix, 1}, {"", "clear", prefix, 2},
		{"", "", prefix + "-literal%_\\", 1}, {"", "", ids[0], 1}, {"", "", input.BookingID, 1},
	} {
		rows, total, err := svc.ListPayments(t.Context(), dto.ListPaymentsQuery{Page: 1, Limit: 20, Search: test.search, Status: test.status, Reconciliation: test.review})
		if err != nil || total != test.total || int64(len(rows)) != test.total {
			t.Fatalf("filter status=%s review=%s total=%d error=%v", test.status, test.review, total, err)
		}
	}
	if provider.calls != 3 {
		t.Fatal("list invoked provider")
	}
}

func TestStaffPaymentDetailTimelinePostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	payment, _, err := svc.CreateVA(t.Context(), input, "create")
	if err != nil {
		t.Fatal(err)
	}
	attempt := reloadAttempt(t, db, payment.PaymentID)
	at := attempt.CreatedAt.Add(time.Minute)
	observationIDs := []string{uuid.NewString(), uuid.NewString()}
	for _, id := range observationIDs {
		if err := db.Exec(`INSERT INTO payment.provider_observations (id, payment_id, source, provider_status, verification_result, error_code, observed_at) VALUES (?, ?, 'webhook', 'pending', 'unavailable', 'PROVIDER_VERIFICATION_UNAVAILABLE', ?)`, id, payment.PaymentID, at).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc.configured = false
	detail, err := svc.PaymentDetail(t.Context(), payment.PaymentID, "detail")
	if err != nil || detail.BookingStatus != "pending_payment" || detail.AmountIDR != 150000 || len(detail.Timeline) != 4 || detail.Timeline[0].Source != "created" {
		t.Fatalf("detail=%+v error=%v", detail, err)
	}
	if detail.Timeline[2].ID > detail.Timeline[3].ID || detail.Timeline[1].Source != "charge" || provider.calls != 1 {
		t.Fatal("timeline ordering or read isolation failed")
	}
	if _, err := svc.PaymentDetail(t.Context(), uuid.NewString(), "missing"); !errors.Is(err, repository.ErrPaymentNotFound) {
		t.Fatalf("missing=%v", err)
	}
	reader.err = booking.ErrNotFound
	if _, err := svc.PaymentDetail(t.Context(), payment.PaymentID, "booking-missing"); !errors.Is(err, booking.ErrUnavailable) {
		t.Fatalf("broken Booking link=%v", err)
	}
}
