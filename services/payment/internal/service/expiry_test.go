package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"payment/internal/booking"
	"payment/internal/midtrans"
	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
)

func TestExpiryClosesUnusedInitiationPostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	before := svc.now()
	svc.now = func() time.Time { return reader.snapshot.HoldExpiresAt }
	first, err := svc.AssessExpiry(t.Context(), input.BookingID, "close")
	if err != nil || first.Result != "no_attempt_closed" || first.PaymentID != nil || first.ClosedAt == nil || !first.ClosedAt.Equal(reader.snapshot.HoldExpiresAt) {
		t.Fatalf("closure=%+v error=%v", first, err)
	}
	restarted := NewPaymentService(repository.NewPaymentRepository(db), reader, provider, true)
	restarted.now = func() time.Time { return reader.snapshot.HoldExpiresAt.Add(time.Minute) }
	retry, err := restarted.AssessExpiry(t.Context(), input.BookingID, "retry")
	if err != nil || retry.ClosedAt == nil || !retry.ClosedAt.Equal(*first.ClosedAt) {
		t.Fatal("retry/restart changed the permanent closure")
	}
	// Simulate a create request that already passed eligibility before expiry.
	svc.now = func() time.Time { return before }
	if _, _, err := svc.CreateVA(t.Context(), input, "delayed-create"); !errors.Is(err, repository.ErrInitiationClosed) {
		t.Fatalf("closed initiation error=%v", err)
	}
	if provider.calls != 0 || countPaymentRows(t, db, "payment_attempts", "booking_id", input.BookingID) != 0 {
		t.Fatal("delayed create reached the provider or persisted an attempt")
	}
}

func TestExpiryEligibilityPostgres(t *testing.T) {
	for _, state := range []string{"before", "confirmed", "expired", "payment_review", "missing", "unavailable"} {
		t.Run(state, func(t *testing.T) {
			svc, db, reader, _, input := paymentFixture(t)
			want := ErrNotExpirable
			if state != "before" {
				svc.now = func() time.Time { return reader.snapshot.HoldExpiresAt.Add(time.Minute) }
			}
			switch state {
			case "missing":
				reader.err, want = booking.ErrNotFound, booking.ErrNotFound
			case "unavailable":
				reader.err, want = booking.ErrUnavailable, booking.ErrUnavailable
			case "before":
			default:
				reader.snapshot.Status = state
			}
			_, err := svc.AssessExpiry(t.Context(), input.BookingID, "ineligible")
			if !errors.Is(err, want) || countPaymentRows(t, db, "initiation_gates", "booking_id", input.BookingID) != 0 {
				t.Fatalf("ineligible request mutated the gate: %v", err)
			}
		})
	}
}

func TestStoredStatusAloneNeverAuthorizesReleasePostgres(t *testing.T) {
	for _, state := range []string{"initiating", "pending", "paid", "expired", "failed"} {
		t.Run(state, func(t *testing.T) {
			svc, db, reader, provider, input := paymentFixture(t)
			attempt, _, err := svc.CreateVA(t.Context(), input, "create")
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("UPDATE payment.payment_attempts SET status = ?, verified_at = ? WHERE id = ?", state, svc.now(), attempt.PaymentID).Error; err != nil {
				t.Fatal(err)
			}
			svc.now = func() time.Time { return reader.snapshot.HoldExpiresAt.Add(time.Minute) }
			result, err := svc.AssessExpiry(t.Context(), input.BookingID, "existing")
			if state != "paid" {
				if !errors.Is(err, ErrVerificationUnavailable) || reloadAttempt(t, db, attempt.PaymentID).ReleaseAuthorizedAt != nil {
					t.Fatalf("unverified expiry authorized release: %+v %v", result, err)
				}
				return
			}
			if err != nil || result.Result != "attempt_exists" || result.ClosedAt != nil || result.PaymentID == nil || *result.PaymentID != attempt.PaymentID || provider.calls != 1 {
				t.Fatalf("attempt assessment=%+v error=%v", result, err)
			}
		})
	}
}

func TestChargeInFlightPreventsClosurePostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	started, finish := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(finish) }) }
	defer unblock()
	provider.afterCharge = func(midtrans.ChargeInput) { close(started); <-finish }
	created := make(chan error, 1)
	go func() { _, _, err := svc.CreateVA(t.Context(), input, "in-flight"); created <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("charge did not start")
	}
	assessor := NewPaymentService(repository.NewPaymentRepository(db), reader, nil, false)
	assessor.now = func() time.Time { return reader.snapshot.HoldExpiresAt }
	result, err := assessor.AssessExpiry(t.Context(), input.BookingID, "expiry")
	if err != nil || result.Result != "attempt_exists" || result.ClosedAt != nil {
		t.Fatalf("in-flight assessment=%+v error=%v", result, err)
	}
	unblock()
	if err := <-created; err != nil {
		t.Fatal(err)
	}
}

func TestReservationRacesClosurePostgres(t *testing.T) {
	for range 8 {
		svc, db, reader, _, input := paymentFixture(t)
		at := reader.snapshot.HoldExpiresAt
		start := make(chan struct{})
		type outcome struct {
			created bool
			state   repository.InitiationState
			err     error
		}
		reserved, assessed := make(chan outcome, 1), make(chan outcome, 1)
		go func() {
			<-start
			_, created, err := svc.repo.Reserve(t.Context(), model.PaymentAttempt{ID: uuid.NewString(), BookingID: input.BookingID, CustomerID: input.CustomerID, ProviderOrderID: uuid.NewString(), AmountIDR: 150000, Currency: "IDR", HoldExpiresAt: at}, input.IdempotencyKey, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			reserved <- outcome{created: created, err: err}
		}()
		go func() {
			<-start
			state, err := svc.repo.CloseUnusedInitiation(t.Context(), input.BookingID, at)
			assessed <- outcome{state: state, err: err}
		}()
		close(start)
		r, a := <-reserved, <-assessed
		if a.err != nil {
			t.Fatal(a.err)
		}
		if r.created {
			if r.err != nil || a.state.ClosedAt != nil || a.state.PaymentID == nil {
				t.Fatal("an existing reservation was authorized for release")
			}
		} else if !errors.Is(r.err, repository.ErrInitiationClosed) || a.state.ClosedAt == nil || a.state.PaymentID != nil {
			t.Fatalf("reservation=%+v assessment=%+v", r, a)
		}
		want := int64(0)
		if r.created {
			want = 1
		}
		if countPaymentRows(t, db, "payment_attempts", "booking_id", input.BookingID) != want {
			t.Fatal("partial reservation persisted")
		}
	}
}

func TestExpiryStaffIndicator(t *testing.T) {
	deadline := time.Date(2030, 1, 1, 0, 15, 0, 0, time.UTC)
	early, late := deadline.Add(-time.Minute), deadline.Add(time.Minute)
	for _, test := range []struct {
		name, status string
		now          time.Time
		verified     *time.Time
		want         bool
	}{
		{"before", "pending", early, nil, false}, {"at deadline", "pending", deadline, nil, true},
		{"initiating", "initiating", late, nil, true}, {"expired", "expired", late, nil, true}, {"failed", "failed", late, nil, true},
		{"early paid", "paid", late, &early, false}, {"paid at deadline", "paid", late, &deadline, true}, {"late paid", "paid", late, &late, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := staffPayment(model.PaymentAttempt{Status: test.status, HoldExpiresAt: deadline, VerifiedAt: test.verified}, test.now)
			if result.ExpiryReviewRequired != test.want || result.ReviewRequired || !result.HoldExpiresAt.Equal(deadline) {
				t.Fatal("operational expiry review changed payment verification facts")
			}
		})
	}
}

func TestVerifiedProviderExpiryAuthorizesReleasePostgres(t *testing.T) {
	for _, initial := range []string{"initiating", "pending", "expired", "failed"} {
		t.Run(initial, func(t *testing.T) {
			svc, db, reader, provider, input := paymentFixture(t)
			created, _, err := svc.CreateVA(t.Context(), input, "create")
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("UPDATE payment.payment_attempts SET status = ? WHERE id = ?", initial, created.PaymentID).Error; err != nil {
				t.Fatal(err)
			}
			attempt := reloadAttempt(t, db, created.PaymentID)
			provider.status = midtrans.StatusResult{OrderID: attempt.ProviderOrderID, TransactionID: *attempt.ProviderTransactionID,
				MerchantID: "test-merchant", Status: "expire", StatusCode: "200", PaymentType: "bank_transfer", Bank: "bni", Currency: "IDR", AmountIDR: attempt.AmountIDR}
			svc.now = func() time.Time { return reader.snapshot.HoldExpiresAt }
			assessment, err := svc.AssessExpiry(t.Context(), input.BookingID, "provider-expiry")
			if err != nil || assessment.Result != "provider_expired" || assessment.PaymentID == nil ||
				*assessment.PaymentID != attempt.ID || assessment.ReleaseAuthorizedAt == nil || !assessment.ReleaseAuthorizedAt.Equal(reader.snapshot.HoldExpiresAt) || assessment.ClosedAt != nil {
				t.Fatalf("assessment=%+v error=%v", assessment, err)
			}
			saved := reloadAttempt(t, db, attempt.ID)
			if saved.Status != "expired" || saved.ReleaseAuthorizedAt == nil || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 0 {
				t.Fatal("expiry permission was not saved, or expiry invented a paid event")
			}
			provider.statusError = midtrans.ErrUnavailable
			retry, err := svc.AssessExpiry(t.Context(), input.BookingID, "retry")
			if err != nil || retry.ReleaseAuthorizedAt == nil || !retry.ReleaseAuthorizedAt.Equal(*assessment.ReleaseAuthorizedAt) ||
				countPaymentRows(t, db, "provider_observations", "payment_id", attempt.ID) != 2 {
				t.Fatal("retry changed permission or repeated the provider lookup")
			}
			// A later paid fact is retained but can never authorize automatic confirmation.
			provider.status.Status, provider.status.FraudStatus = "settlement", "accept"
			observation := providerObservation(attempt.ID, provider.status, "status_lookup", "late-paid")
			outcome := verifiedPaymentOutcome(attempt, provider.status, observation, svc.now().Add(time.Second))
			if err := svc.repo.ApplyVerifiedObservation(t.Context(), attempt.ID, "", outcome); err != nil {
				t.Fatal(err)
			}
			saved = reloadAttempt(t, db, attempt.ID)
			if saved.Status != "paid" || saved.ReviewReason == nil || *saved.ReviewReason != "PAID_AFTER_RELEASE_AUTHORIZATION" ||
				saved.ReleaseAuthorizedAt == nil || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 1 {
				t.Fatal("late payment was lost or did not require review")
			}
		})
	}
}

func TestProviderExpiryRequiresMatchingFactsPostgres(t *testing.T) {
	for _, scenario := range []string{"pending", "deny", "settlement", "mismatch", "unavailable", "review"} {
		t.Run(scenario, func(t *testing.T) {
			svc, db, reader, provider, input := paymentFixture(t)
			created, _, err := svc.CreateVA(t.Context(), input, "create")
			if err != nil {
				t.Fatal(err)
			}
			attempt := reloadAttempt(t, db, created.PaymentID)
			provider.status = midtrans.StatusResult{OrderID: attempt.ProviderOrderID, TransactionID: *attempt.ProviderTransactionID,
				MerchantID: "test-merchant", Status: "expire", StatusCode: "200", FraudStatus: "accept", PaymentType: "bank_transfer", Bank: "bni", Currency: "IDR", AmountIDR: attempt.AmountIDR}
			switch scenario {
			case "pending", "deny", "settlement":
				provider.status.Status = scenario
			case "mismatch":
				provider.status.AmountIDR++
			case "unavailable":
				provider.statusError = midtrans.ErrUnavailable
			case "review":
				if err := db.Exec("UPDATE payment.payment_attempts SET review_reason = 'PROVIDER_REFERENCE_MISMATCH' WHERE id = ?", attempt.ID).Error; err != nil {
					t.Fatal(err)
				}
			}
			svc.now = func() time.Time { return reader.snapshot.HoldExpiresAt }
			assessment, err := svc.AssessExpiry(t.Context(), input.BookingID, "guard")
			if scenario == "unavailable" {
				if !errors.Is(err, ErrVerificationUnavailable) {
					t.Fatal("provider failure was not reported")
				}
			} else if err != nil || assessment.Result != "attempt_exists" {
				t.Fatalf("unexpected assessment=%+v error=%v", assessment, err)
			}
			saved := reloadAttempt(t, db, attempt.ID)
			if saved.ReleaseAuthorizedAt != nil || assessment.ReleaseAuthorizedAt != nil {
				t.Fatal("unverified expiry authorized release")
			}
			if scenario == "settlement" && saved.Status != "paid" {
				t.Fatal("expiry lookup discarded a paid fact")
			}
		})
	}
}

func TestPaidAndReleasePermissionUseCurrentStoredStatePostgres(t *testing.T) {
	for _, paidFirst := range []bool{true, false} {
		svc, db, reader, _, input := paymentFixture(t)
		created, _, err := svc.CreateVA(t.Context(), input, "create")
		if err != nil {
			t.Fatal(err)
		}
		attempt := reloadAttempt(t, db, created.PaymentID)
		facts := midtrans.StatusResult{OrderID: attempt.ProviderOrderID, TransactionID: *attempt.ProviderTransactionID,
			MerchantID: "test-merchant", Status: "settlement", StatusCode: "200", FraudStatus: "accept", PaymentType: "bank_transfer", Bank: "bni", Currency: "IDR", AmountIDR: attempt.AmountIDR}
		paid := verifiedPaymentOutcome(attempt, facts, providerObservation(attempt.ID, facts, "status_lookup", "paid"), reader.snapshot.HoldExpiresAt.Add(-time.Second))
		facts.Status = "expire"
		expired := verifiedPaymentOutcome(attempt, facts, providerObservation(attempt.ID, facts, "status_lookup", "expire"), reader.snapshot.HoldExpiresAt)
		expired.AuthorizeRelease = true
		outcomes := []model.PaymentOutcome{expired, paid}
		if paidFirst {
			outcomes = []model.PaymentOutcome{paid, expired}
		}
		for _, outcome := range outcomes {
			if err := svc.repo.ApplyVerifiedObservation(t.Context(), attempt.ID, "", outcome); err != nil {
				t.Fatal(err)
			}
		}
		saved := reloadAttempt(t, db, attempt.ID)
		if saved.Status != "paid" || (paidFirst && saved.ReleaseAuthorizedAt != nil) ||
			(!paidFirst && (saved.ReleaseAuthorizedAt == nil || saved.ReviewReason == nil || *saved.ReviewReason != "PAID_AFTER_RELEASE_AUTHORIZATION")) {
			t.Fatal("stale observations bypassed the persisted paid/release decision")
		}
	}
}
