package service

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"payment/internal/booking"
	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/repository"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testBookingReader struct {
	snapshot booking.PaymentContext
	err      error
}

func (b *testBookingReader) PaymentContext(context.Context, string, string) (booking.PaymentContext, error) {
	return b.snapshot, b.err
}

type testProvider struct {
	mu          sync.Mutex
	calls       int
	err         error
	afterCharge func(midtrans.ChargeInput)
}

func (p *testProvider) CreateVA(_ context.Context, input midtrans.ChargeInput, _ string) (midtrans.ChargeResult, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.afterCharge != nil {
		p.afterCharge(input)
	}
	return midtrans.ChargeResult{OrderID: input.OrderID, TransactionID: input.OrderID, Status: "pending", AmountIDR: input.AmountIDR,
		Currency: "IDR", Bank: "bni", VANumber: "0000000000000000", ExpiresAt: input.HoldExpiresAt.Truncate(time.Second)}, p.err
}

func paymentFixture(t *testing.T) (*PaymentService, *gorm.DB, *testBookingReader, *testProvider, dto.CreateVARequest) {
	t.Helper()
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
			t.Error("could not close test pool")
		}
	})
	input := dto.CreateVARequest{CustomerID: uuid.NewString(), BookingID: uuid.NewString(), IdempotencyKey: uuid.NewString()}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, child := range []struct{ table, column string }{
			{"outbox_events", "aggregate_id"}, {"provider_observations", "payment_id"},
			{"webhook_receipts", "payment_id"}, {"idempotency_records", "payment_id"},
		} {
			if err := db.WithContext(ctx).Exec("DELETE FROM payment."+child.table+" WHERE "+child.column+" IN (SELECT id FROM payment.payment_attempts WHERE customer_id = ?)", input.CustomerID).Error; err != nil {
				t.Error("could not clean test child rows")
			}
		}
		if err := db.WithContext(ctx).Exec("DELETE FROM payment.payment_attempts WHERE customer_id = ?", input.CustomerID).Error; err != nil {
			t.Error("could not clean test attempts")
		}
		if err := db.WithContext(ctx).Exec("DELETE FROM payment.initiation_gates WHERE booking_id = ?", input.BookingID).Error; err != nil {
			t.Error("could not clean test initiation gate")
		}
	})
	now := time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)
	reader := &testBookingReader{snapshot: booking.PaymentContext{BookingID: input.BookingID, CustomerID: input.CustomerID, TotalIDR: 150000, Currency: "IDR", Status: "pending_payment", HoldExpiresAt: now.Add(15 * time.Minute)}}
	provider := &testProvider{}
	svc := NewPaymentService(repository.NewPaymentRepository(db), reader, provider, true)
	svc.now = func() time.Time { return now }
	return svc, db, reader, provider, input
}

func TestCreateAndRetryPostgres(t *testing.T) {
	svc, db, _, provider, input := paymentFixture(t)
	first, created, err := svc.CreateVA(t.Context(), input, "first")
	if err != nil || !created || first.Status != "pending" || first.VANumber == nil || first.AmountIDR != 150000 {
		t.Fatalf("create: status=%s created=%v error=%v", first.Status, created, err)
	}
	for i := range 2 {
		if i == 1 {
			input.IdempotencyKey = uuid.NewString()
		}
		retry, created, err := svc.CreateVA(t.Context(), input, "retry")
		if err != nil || created || retry.PaymentID != first.PaymentID || !retry.HoldExpiresAt.Equal(first.HoldExpiresAt) {
			t.Fatalf("retry: created=%v error=%v", created, err)
		}
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls=%d", provider.calls)
	}
	var keys int64
	if err := db.Table("payment.idempotency_records").Where("payment_id = ?", first.PaymentID).Count(&keys).Error; err != nil {
		t.Fatal(err)
	}
	if keys != 2 {
		t.Fatalf("saved keys=%d", keys)
	}
}

func TestEligibilityPostgres(t *testing.T) {
	for _, name := range []string{"other customer", "missing", "invalid snapshot", "confirmed", "expired", "short window", "not configured"} {
		t.Run(name, func(t *testing.T) {
			svc, _, reader, provider, input := paymentFixture(t)
			want := ErrNotPayable
			switch name {
			case "other customer":
				reader.snapshot.CustomerID = uuid.NewString()
				want = booking.ErrNotFound
			case "missing":
				reader.err = booking.ErrNotFound
				want = booking.ErrNotFound
			case "invalid snapshot":
				reader.snapshot.TotalIDR = 0
				want = booking.ErrUnavailable
			case "confirmed":
				reader.snapshot.Status = "confirmed"
			case "expired":
				reader.snapshot.HoldExpiresAt = svc.now()
				want = ErrHoldExpired
			case "short window":
				reader.snapshot.HoldExpiresAt = svc.now().Add(19 * time.Second)
				want = ErrWindowTooShort
			case "not configured":
				svc.configured = false
				want = ErrNotConfigured
			}
			_, _, err := svc.CreateVA(t.Context(), input, "eligibility")
			if !errors.Is(err, want) || provider.calls != 0 {
				t.Fatalf("error=%v calls=%d", err, provider.calls)
			}
		})
	}
}

func TestKeyConflictRollsBackNewAttemptPostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	if _, _, err := svc.CreateVA(t.Context(), input, "first"); err != nil {
		t.Fatal(err)
	}
	input.BookingID = uuid.NewString()
	reader.snapshot.BookingID = input.BookingID
	_, _, err := svc.CreateVA(t.Context(), input, "conflict")
	if !errors.Is(err, repository.ErrIdempotencyConflict) || provider.calls != 1 {
		t.Fatalf("error=%v calls=%d", err, provider.calls)
	}
	var count int64
	if err := db.Table("payment.payment_attempts").Where("booking_id = ?", input.BookingID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("conflicting key left an orphan attempt")
	}
}

func TestUncertainChargeAndRestartPostgres(t *testing.T) {
	for _, providerError := range []error{midtrans.ErrUnavailable, midtrans.ErrInvalidResponse} {
		t.Run(providerError.Error(), func(t *testing.T) {
			svc, db, reader, provider, input := paymentFixture(t)
			provider.err = providerError
			first, _, err := svc.CreateVA(t.Context(), input, "uncertain")
			if err != nil || first.Status != "initiating" || first.VANumber != nil || first.ReviewRequired != errors.Is(providerError, midtrans.ErrInvalidResponse) {
				t.Fatalf("uncertain charge: status=%s error=%v", first.Status, err)
			}
			restarted := NewPaymentService(repository.NewPaymentRepository(db), reader, provider, true)
			restarted.now = svc.now
			retry, _, err := restarted.CreateVA(t.Context(), input, "restart")
			if err != nil || retry.PaymentID != first.PaymentID || provider.calls != 1 {
				t.Fatalf("restart error=%v calls=%d", err, provider.calls)
			}
		})
	}
}

func TestConcurrentCreatePostgres(t *testing.T) {
	svc, db, _, provider, input := paymentFixture(t)
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := range 2 {
		go func(i int) {
			request := input
			if i == 1 {
				request.IdempotencyKey = uuid.NewString()
			}
			<-start
			_, _, err := svc.CreateVA(t.Context(), request, "parallel")
			results <- err
		}(i)
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Table("payment.payment_attempts").Where("booking_id = ?", input.BookingID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 || provider.calls != 1 {
		t.Fatalf("attempts=%d provider calls=%d", count, provider.calls)
	}
}

func TestPersistenceFailureAfterChargePostgres(t *testing.T) {
	svc, db, reader, provider, input := paymentFixture(t)
	provider.afterCharge = func(charge midtrans.ChargeInput) {
		err := db.Exec(`INSERT INTO payment.payment_attempts
            (booking_id, customer_id, provider_order_id, provider_transaction_id, amount_idr, hold_expires_at)
            VALUES (gen_random_uuid(), ?, gen_random_uuid()::text, ?, 150000, NOW() + INTERVAL '15 minutes')`, input.CustomerID, charge.OrderID).Error
		if err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := svc.CreateVA(t.Context(), input, "save-failure")
	if !errors.Is(err, repository.ErrUnavailable) {
		t.Fatalf("error=%v", err)
	}
	restarted := NewPaymentService(repository.NewPaymentRepository(db), reader, provider, true)
	restarted.now = svc.now
	retry, _, err := restarted.CreateVA(t.Context(), input, "retry")
	if err != nil || retry.Status != "initiating" || retry.VANumber != nil || provider.calls != 1 {
		t.Fatalf("retry status=%s error=%v calls=%d", retry.Status, err, provider.calls)
	}
}
