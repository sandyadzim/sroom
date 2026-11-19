package service

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"payment/internal/dto"
	"payment/internal/midtrans"
	"payment/internal/model"
	"payment/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const webhookTestKey = "synthetic-webhook-key"

type statusProvider struct {
	mu          sync.Mutex
	result      midtrans.StatusResult
	err         error
	calls       int
	orders      []string
	afterLookup func()
}

func (p *statusProvider) GetStatus(_ context.Context, orderID, _ string) (midtrans.StatusResult, error) {
	p.mu.Lock()
	p.calls++
	p.orders = append(p.orders, orderID)
	p.mu.Unlock()
	if p.afterLookup != nil {
		p.afterLookup()
	}
	return p.result, p.err
}

func signedNotification(orderID string) dto.MidtransNotification {
	digest := sha512.Sum512([]byte(orderID + "200" + "150000.00" + webhookTestKey))
	return dto.MidtransNotification{OrderID: orderID, StatusCode: "200", GrossAmount: "150000.00", SignatureKey: hex.EncodeToString(digest[:])}
}

func notificationHash(notification dto.MidtransNotification) string {
	body, _ := json.Marshal(notification)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func webhookFixture(t *testing.T) (*WebhookService, *gorm.DB, *statusProvider, model.PaymentAttempt, dto.MidtransNotification) {
	t.Helper()
	paymentService, db, _, _, input := paymentFixture(t)
	payment, _, err := paymentService.CreateVA(t.Context(), input, "create")
	if err != nil {
		t.Fatal(err)
	}
	var attempt model.PaymentAttempt
	if err := db.Table("payment.payment_attempts").Where("id = ?", payment.PaymentID).Take(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	provider := &statusProvider{result: midtrans.StatusResult{OrderID: attempt.ProviderOrderID, TransactionID: *attempt.ProviderTransactionID,
		MerchantID: "synthetic-merchant", Status: "settlement", StatusCode: "200", PaymentType: "bank_transfer", Currency: "IDR", Bank: "bni", FraudStatus: "accept", AmountIDR: 150000}}
	service := NewWebhookService(repository.NewPaymentRepository(db), provider, webhookTestKey)
	service.now = paymentService.now
	return service, db, provider, attempt, signedNotification(attempt.ProviderOrderID)
}

func reloadAttempt(t *testing.T, db *gorm.DB, id string) model.PaymentAttempt {
	t.Helper()
	var attempt model.PaymentAttempt
	if err := db.Table("payment.payment_attempts").Where("id = ?", id).Take(&attempt).Error; err != nil {
		t.Fatal(err)
	}
	return attempt
}

func countPaymentRows(t *testing.T, db *gorm.DB, table, column, id string) int64 {
	t.Helper()
	var count int64
	if err := db.Table("payment."+table).Where(column+" = ?", id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestWebhookPaidAndDuplicatesPostgres(t *testing.T) {
	svc, db, provider, attempt, notification := webhookFixture(t)
	const deliveries = 8
	var wg sync.WaitGroup
	results := make(chan error, deliveries)
	for range deliveries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- svc.Process(t.Context(), notification, notificationHash(notification), "paid-test")
		}()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	paid := reloadAttempt(t, db, attempt.ID)
	if paid.Status != "paid" || paid.VerifiedAt == nil || paid.ProviderPaidAt != nil {
		t.Fatalf("unexpected paid state: %+v", paid)
	}
	firstVerified := *paid.VerifiedAt
	// A different receipt reporting stale pending still uses the canonical GET result.
	notification.Status = "pending"
	svc.now = func() time.Time { return firstVerified.Add(time.Minute) }
	if err := svc.Process(t.Context(), notification, notificationHash(notification), "different-receipt"); err != nil {
		t.Fatal(err)
	}
	paid = reloadAttempt(t, db, attempt.ID)
	if !paid.VerifiedAt.Equal(firstVerified) || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 1 || countPaymentRows(t, db, "webhook_receipts", "payment_id", attempt.ID) != 2 {
		t.Fatal("duplicate changed the paid fact/event")
	}
	var envelope string
	if err := db.Raw("SELECT envelope::text FROM payment.outbox_events WHERE aggregate_id = ?", attempt.ID).Scan(&envelope).Error; err != nil {
		t.Fatal(err)
	}
	var event struct {
		EventType string `json:"event_type"`
		Payload   struct {
			PaymentID      string     `json:"payment_id"`
			BookingID      string     `json:"booking_id"`
			Amount         int64      `json:"amount_idr"`
			ProviderPaidAt *time.Time `json:"provider_paid_at"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(envelope), &event); err != nil {
		t.Fatal(err)
	}
	if event.EventType != "payment.paid.v1" || event.Payload.PaymentID != attempt.ID || event.Payload.BookingID != attempt.BookingID || event.Payload.Amount != 150000 || event.Payload.ProviderPaidAt != nil {
		t.Fatal("wrong paid event")
	}
	provider.result.Status = "expire"
	notification.Status = "expire"
	if err := svc.Process(t.Context(), notification, notificationHash(notification), "late-expiry"); err != nil {
		t.Fatal(err)
	}
	paid = reloadAttempt(t, db, attempt.ID)
	if paid.Status != "paid" || !paid.VerifiedAt.Equal(firstVerified) || paid.ReviewReason == nil || *paid.ReviewReason != "PROVIDER_STATUS_CONTRADICTION" {
		t.Fatal("paid was overwritten by stale expiry")
	}
}

func TestWebhookInvalidAndUnmatchedPostgres(t *testing.T) {
	svc, db, provider, attempt, notification := webhookFixture(t)
	notification.SignatureKey = "invalid"
	if err := svc.Process(t.Context(), notification, notificationHash(notification), "bad-signature"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("error=%v", err)
	}
	if provider.calls != 0 || countPaymentRows(t, db, "webhook_receipts", "payment_id", attempt.ID) != 0 {
		t.Fatal("invalid signature caused side effects")
	}
	notification = signedNotification("unknown-" + uuid.NewString())
	hash := notificationHash(notification)
	t.Cleanup(func() {
		if err := db.Exec("DELETE FROM payment.webhook_receipts WHERE payload_hash = ?", hash).Error; err != nil {
			t.Error(err)
		}
	})
	if err := svc.Process(t.Context(), notification, hash, "unknown"); err != nil {
		t.Fatal(err)
	}
	var receipt model.WebhookReceipt
	if err := db.Table("payment.webhook_receipts").Where("payload_hash = ?", hash).Take(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "unmatched" || receipt.PaymentID != nil || provider.calls != 0 {
		t.Fatal("unknown order was not safely acknowledged")
	}
}

func TestWebhookUnavailableRetryPostgres(t *testing.T) {
	svc, db, provider, attempt, notification := webhookFixture(t)
	provider.err = midtrans.ErrUnavailable
	hash := notificationHash(notification)
	if err := svc.Process(t.Context(), notification, hash, "unavailable"); !errors.Is(err, ErrVerificationUnavailable) {
		t.Fatalf("error=%v", err)
	}
	var receipt model.WebhookReceipt
	if err := db.Table("payment.webhook_receipts").Where("payload_hash = ?", hash).Take(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "received" || reloadAttempt(t, db, attempt.ID).Status != "pending" {
		t.Fatal("temporary failure acknowledged or marked paid")
	}
	provider.err = nil
	if err := svc.Process(t.Context(), notification, hash, "retry"); err != nil {
		t.Fatal(err)
	}
	if reloadAttempt(t, db, attempt.ID).Status != "paid" {
		t.Fatal("retry did not recover")
	}
}

func TestWebhookRejectedLookupsPostgres(t *testing.T) {
	for _, name := range []string{"order", "transaction", "amount", "currency", "bank", "channel", "fraud", "unknown status", "invalid response", "notification amount", "merchant", "reference changed during lookup"} {
		t.Run(name, func(t *testing.T) {
			svc, db, provider, attempt, notification := webhookFixture(t)
			switch name {
			case "order":
				provider.result.OrderID = "another-order"
			case "transaction":
				provider.result.TransactionID = "another-transaction"
			case "amount":
				provider.result.AmountIDR++
			case "currency":
				provider.result.Currency = "USD"
			case "bank":
				provider.result.Bank = "bca"
			case "channel":
				provider.result.PaymentType = "credit_card"
			case "fraud":
				provider.result.FraudStatus = "challenge"
			case "unknown status":
				provider.result.Status = "refund"
			case "invalid response":
				provider.err = midtrans.ErrInvalidResponse
			case "notification amount":
				notification.GrossAmount = "150001.00"
				digest := sha512.Sum512([]byte(notification.OrderID + notification.StatusCode + notification.GrossAmount + webhookTestKey))
				notification.SignatureKey = hex.EncodeToString(digest[:])
			case "merchant":
				notification.MerchantID = "another-merchant"
			case "reference changed during lookup":
				provider.afterLookup = func() {
					if err := db.Exec("UPDATE payment.payment_attempts SET provider_transaction_id = ? WHERE id = ?", "changed-"+uuid.NewString(), attempt.ID).Error; err != nil {
						t.Error(err)
					}
				}
			}
			if err := svc.Process(t.Context(), notification, notificationHash(notification), "mismatch"); err != nil {
				t.Fatal(err)
			}
			updated := reloadAttempt(t, db, attempt.ID)
			if updated.Status != "pending" || updated.ReviewReason == nil || updated.VerifiedAt != nil || countPaymentRows(t, db, "outbox_events", "aggregate_id", attempt.ID) != 0 {
				t.Fatalf("mismatch changed paid state: %+v", updated)
			}
			var receipt model.WebhookReceipt
			if err := db.Table("payment.webhook_receipts").Where("payload_hash = ?", notificationHash(notification)).Take(&receipt).Error; err != nil {
				t.Fatal(err)
			}
			if receipt.Status != "rejected" {
				t.Fatal("mismatch not durably rejected")
			}
		})
	}
}

func TestWebhookTransitionsPostgres(t *testing.T) {
	for _, status := range []string{"pending", "expire", "deny", "cancel", "failure"} {
		t.Run(status, func(t *testing.T) {
			svc, db, provider, attempt, notification := webhookFixture(t)
			provider.result.Status = status
			if err := svc.Process(t.Context(), notification, notificationHash(notification), "transition"); err != nil {
				t.Fatal(err)
			}
			want := "failed"
			if status == "pending" {
				want = "pending"
			}
			if status == "expire" {
				want = "expired"
			}
			if reloadAttempt(t, db, attempt.ID).Status != want {
				t.Fatalf("want=%s", want)
			}
			// Verified funds remain facts even after a terminal state or hold deadline.
			provider.result.Status = "settlement"
			notification.Status = "settlement"
			svc.now = func() time.Time { return attempt.HoldExpiresAt.Add(time.Hour) }
			if err := svc.Process(t.Context(), notification, notificationHash(notification), "late-paid"); err != nil {
				t.Fatal(err)
			}
			paid := reloadAttempt(t, db, attempt.ID)
			if paid.Status != "paid" || paid.VerifiedAt == nil {
				t.Fatal("late paid fact discarded")
			}
			if want != "pending" && (paid.ReviewReason == nil || *paid.ReviewReason != "PAID_AFTER_TERMINAL_STATUS") {
				t.Fatal("missing terminal review")
			}
		})
	}
}

func TestWebhookTransactionRollbackPostgres(t *testing.T) {
	svc, db, _, attempt, notification := webhookFixture(t)
	// Force an outbox uniqueness failure. All effects in that transaction must roll back.
	if err := db.Exec(`INSERT INTO payment.outbox_events (id, aggregate_id, event_type, envelope, occurred_at) VALUES (?, ?, 'payment.paid.v1', '{}', NOW())`, uuid.NewString(), attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Process(t.Context(), notification, notificationHash(notification), "rollback"); !errors.Is(err, repository.ErrUnavailable) {
		t.Fatalf("error=%v", err)
	}
	if reloadAttempt(t, db, attempt.ID).Status != "pending" || countPaymentRows(t, db, "provider_observations", "payment_id", attempt.ID) != 1 {
		t.Fatal("failed transaction left partial effects")
	}
	var receipt model.WebhookReceipt
	if err := db.Table("payment.webhook_receipts").Where("payload_hash = ?", notificationHash(notification)).Take(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "received" {
		t.Fatal("failed transaction acknowledged")
	}
	if err := db.Exec("DELETE FROM payment.outbox_events WHERE aggregate_id = ?", attempt.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Process(t.Context(), notification, notificationHash(notification), "retry"); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookBeforeChargePersistencePostgres(t *testing.T) {
	paymentService, db, _, chargeProvider, input := paymentFixture(t)
	chargeProvider.afterCharge = func(input midtrans.ChargeInput) {
		provider := &statusProvider{result: midtrans.StatusResult{OrderID: input.OrderID, TransactionID: input.OrderID, MerchantID: "synthetic-merchant", Status: "settlement", StatusCode: "200", FraudStatus: "accept", AmountIDR: input.AmountIDR, Currency: "IDR", Bank: "bni", PaymentType: "bank_transfer"}}
		svc := NewWebhookService(repository.NewPaymentRepository(db), provider, webhookTestKey)
		svc.now = paymentService.now
		notification := signedNotification(input.OrderID)
		if err := svc.Process(t.Context(), notification, notificationHash(notification), "early-webhook"); err != nil {
			t.Error(err)
		}
	}
	payment, _, err := paymentService.CreateVA(t.Context(), input, "charge")
	if err != nil || payment.Status != "paid" {
		t.Fatalf("charge overwrote webhook: status=%s error=%v", payment.Status, err)
	}
	if countPaymentRows(t, db, "outbox_events", "aggregate_id", payment.PaymentID) != 1 {
		t.Fatal("paid event missing")
	}
}
