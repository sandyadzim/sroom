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
	"gorm.io/gorm"
)

func refreshFixture(t *testing.T) (*RefreshPaymentService, *gorm.DB, *testBookingReader, *statusProvider, model.PaymentAttempt) {
	t.Helper()
	payments, db, reader, _, input := paymentFixture(t)
	created, _, err := payments.CreateVA(t.Context(), input, "create")
	if err != nil {
		t.Fatal(err)
	}
	attempt := reloadAttempt(t, db, created.PaymentID)
	provider := &statusProvider{result: midtrans.StatusResult{OrderID: attempt.ProviderOrderID, TransactionID: *attempt.ProviderTransactionID,
		MerchantID: "synthetic-merchant", Status: "settlement", StatusCode: "200", PaymentType: "bank_transfer", Currency: "IDR", Bank: "bni", FraudStatus: "accept", AmountIDR: 150000}}
	return NewRefreshPaymentService(payments, provider, true), db, reader, provider, attempt
}

func TestRefreshPendingPaidAndRetryPostgres(t *testing.T) {
	svc, db, _, provider, attempt := refreshFixture(t)
	provider.result.Status = "pending"
	pending, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "refresh-pending")
	if err != nil || pending.Status != "pending" || pending.VANumber == nil || pending.ReviewRequired {
		t.Fatalf("pending=%+v error=%v", pending, err)
	}
	provider.result.Status = "settlement"
	paid, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "refresh-paid")
	if err != nil || paid.Status != "paid" || paid.VANumber != nil || paid.VerifiedAt == nil || paid.ProviderPaidAt != nil || paid.BookingStatus != "pending_payment" {
		t.Fatalf("paid=%+v error=%v", paid, err)
	}
	svc.configured = false
	provider.err = midtrans.ErrUnavailable
	retry, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "retry-paid")
	if err != nil || retry.PaymentID == nil || *retry.PaymentID != attempt.ID || !retry.VerifiedAt.Equal(*paid.VerifiedAt) || provider.calls != 2 {
		t.Fatal("paid retry required provider/key or changed facts")
	}
	if !retry.HoldExpiresAt.Equal(attempt.HoldExpiresAt) || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 1 || countPaymentRows(t, db, "provider_observations", "payment_id", attempt.ID) != 3 || countPaymentRows(t, db, "idempotency_records", "payment_id", attempt.ID) != 1 {
		t.Fatal("refresh duplicated effects or changed deadline")
	}
	var observation model.ProviderObservation
	if err := db.Table("payment.provider_observations").Where("payment_id = ? AND request_id = ?", attempt.ID, "refresh-paid").Take(&observation).Error; err != nil {
		t.Fatal(err)
	}
	if observation.Source != "status_lookup" || observation.VerificationResult != "verified" {
		t.Fatal("refresh observation not recorded")
	}
	var event struct{ Cause *string }
	if err := db.Raw("SELECT envelope->>'causation_id' AS cause FROM payment.outbox_events WHERE aggregate_id = ?", attempt.ID).Scan(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.Cause != nil {
		t.Fatal("refresh invented a webhook receipt")
	}
	for _, orderID := range provider.orders {
		if orderID != attempt.ProviderOrderID {
			t.Fatal("refresh used a replacement order ID")
		}
	}
}

func TestRefreshNoAttemptAndAccessPostgres(t *testing.T) {
	payments, db, reader, _, input := paymentFixture(t)
	provider := &statusProvider{}
	svc := NewRefreshPaymentService(payments, provider, false)
	result, err := svc.Refresh(t.Context(), input.BookingID, input.CustomerID, "not-started")
	if err != nil || result.Status != "not_started" || result.PaymentID != nil || provider.calls != 0 || countPaymentRows(t, db, "payment_attempts", "customer_id", input.CustomerID) != 0 {
		t.Fatal("refresh without attempt created payment or required key")
	}
	reader.snapshot.CustomerID = uuid.NewString()
	if _, err := svc.Refresh(t.Context(), input.BookingID, input.CustomerID, "other-owner"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("ownership error=%v", err)
	}
	reader.err = booking.ErrNotFound
	if _, err := svc.Refresh(t.Context(), input.BookingID, input.CustomerID, "missing"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatalf("missing error=%v", err)
	}
	reader.err = booking.ErrUnavailable
	if _, err := svc.Refresh(t.Context(), input.BookingID, input.CustomerID, "booking-down"); !errors.Is(err, booking.ErrUnavailable) {
		t.Fatalf("dependency error=%v", err)
	}
	if provider.calls != 0 {
		t.Fatal("provider called before ownership was established")
	}
}

func TestRefreshExistingOwnerAndKeyPostgres(t *testing.T) {
	svc, db, reader, provider, attempt := refreshFixture(t)
	if _, err := svc.Refresh(t.Context(), attempt.BookingID, uuid.NewString(), "other-customer"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal("cross-customer attempt exposed")
	}
	reader.snapshot.CustomerID = uuid.NewString()
	if _, err := svc.Refresh(t.Context(), attempt.BookingID, reader.snapshot.CustomerID, "stored-owner-mismatch"); !errors.Is(err, booking.ErrNotFound) {
		t.Fatal("stored Payment ownership ignored")
	}
	reader.snapshot.CustomerID = attempt.CustomerID
	svc.configured = false
	if _, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "no-key"); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("configuration error=%v", err)
	}
	if provider.calls != 0 || countPaymentRows(t, db, "provider_observations", "payment_id", attempt.ID) != 1 {
		t.Fatal("denied requests caused provider effects")
	}
}

func TestRefreshUnavailableAndReviewPostgres(t *testing.T) {
	for _, name := range []string{"unavailable", "invalid response", "amount mismatch", "transaction mismatch", "unknown status"} {
		t.Run(name, func(t *testing.T) {
			svc, db, _, provider, attempt := refreshFixture(t)
			switch name {
			case "unavailable":
				provider.err = midtrans.ErrUnavailable
			case "invalid response":
				provider.err = midtrans.ErrInvalidResponse
			case "amount mismatch":
				provider.result.AmountIDR++
			case "transaction mismatch":
				provider.result.TransactionID = "another-transaction"
			case "unknown status":
				provider.result.Status = "unknown"
			}
			result, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "refresh-error")
			saved := reloadAttempt(t, db, attempt.ID)
			if saved.Status != "pending" || saved.VerifiedAt != nil || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 0 {
				t.Fatal("uncertain result treated as payment fact")
			}
			var observation model.ProviderObservation
			if err := db.Table("payment.provider_observations").Where("payment_id = ? AND request_id = ?", attempt.ID, "refresh-error").Take(&observation).Error; err != nil {
				t.Fatal(err)
			}
			if name == "unavailable" {
				if !errors.Is(err, ErrVerificationUnavailable) || observation.VerificationResult != "unavailable" || saved.ReviewReason != nil {
					t.Fatalf("unavailable error=%v", err)
				}
				provider.err = nil
				if _, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "retry"); err != nil {
					t.Fatal(err)
				}
				if reloadAttempt(t, db, attempt.ID).Status != "paid" {
					t.Fatal("manual retry did not recover")
				}
			} else if err != nil || !result.ReviewRequired || result.VANumber != nil || observation.VerificationResult != "rejected" || saved.ReviewReason == nil {
				t.Fatalf("review result=%+v error=%v", result, err)
			}
		})
	}
}

func TestRefreshTransitionsAndLatePaidPostgres(t *testing.T) {
	for _, state := range []string{"expire", "deny", "cancel", "failure"} {
		t.Run(state, func(t *testing.T) {
			svc, db, _, provider, attempt := refreshFixture(t)
			provider.result.Status = state
			result, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "terminal")
			want := "failed"
			if state == "expire" {
				want = "expired"
			}
			if err != nil || result.Status != want {
				t.Fatalf("terminal=%s error=%v", result.Status, err)
			}
			provider.result.Status = "settlement"
			svc.payments.now = func() time.Time { return attempt.HoldExpiresAt.Add(time.Hour) }
			result, err = svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "late-paid")
			saved := reloadAttempt(t, db, attempt.ID)
			if err != nil || result.Status != "paid" || !result.ReviewRequired || saved.ReviewReason == nil || *saved.ReviewReason != "PAID_AFTER_TERMINAL_STATUS" {
				t.Fatal("late funds were lost or not flagged for review")
			}
		})
	}
}

func TestRefreshInitiatingKeepsMissingInstructionsPostgres(t *testing.T) {
	svc, db, _, provider, attempt := refreshFixture(t)
	if err := db.Exec("UPDATE payment.payment_attempts SET status = 'initiating', provider_transaction_id = NULL, va_number = NULL, provider_expires_at = NULL WHERE id = ?", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	provider.result.Status = "pending"
	result, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "recover-status")
	if err != nil || result.Status != "pending" || result.VANumber != nil || result.ProviderExpiresAt != nil || !result.HoldExpiresAt.Equal(attempt.HoldExpiresAt) {
		t.Fatal("refresh invented VA instructions or changed deadline")
	}
	if countPaymentRows(t, db, "payment_attempts", "booking_id", attempt.BookingID) != 1 {
		t.Fatal("refresh replaced the attempt")
	}
}

func TestRefreshAndWebhookConcurrentPostgres(t *testing.T) {
	svc, db, _, provider, attempt := refreshFixture(t)
	webhook := NewWebhookService(repository.NewPaymentRepository(db), provider, webhookTestKey)
	webhook.now = svc.payments.now
	notification := signedNotification(attempt.ProviderOrderID)
	var wg sync.WaitGroup
	results := make(chan error, 8)
	start := make(chan struct{})
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				_, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "parallel-refresh")
				results <- err
			} else {
				results <- webhook.Process(t.Context(), notification, notificationHash(notification), "parallel-webhook")
			}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if reloadAttempt(t, db, attempt.ID).Status != "paid" || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 1 {
		t.Fatal("concurrent delivery duplicated or lost paid effects")
	}
}

func TestRefreshTransactionFailurePostgres(t *testing.T) {
	svc, db, _, _, attempt := refreshFixture(t)
	if err := db.Exec(`INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (?, ?, 'payment.paid.v1', '{}', NOW())`, uuid.NewString(), attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(t.Context(), attempt.BookingID, attempt.CustomerID, "rollback"); !errors.Is(err, repository.ErrUnavailable) {
		t.Fatalf("database error=%v", err)
	}
	if reloadAttempt(t, db, attempt.ID).Status != "pending" || countPaymentRows(t, db, "provider_observations", "payment_id", attempt.ID) != 1 {
		t.Fatal("refresh committed partial payment effects")
	}
}
