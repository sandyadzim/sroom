package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"payment/internal/mailer"
	"payment/internal/model"
	"payment/internal/repository"
)

type emailRecorder struct {
	mu        sync.Mutex
	messages  []mailer.Message
	err       error
	afterSend func()
}

func (r *emailRecorder) Send(_ context.Context, message mailer.Message) error {
	r.mu.Lock()
	r.messages = append(r.messages, message)
	r.mu.Unlock()
	if r.afterSend != nil {
		r.afterSend()
	}
	return r.err
}

func emailInvoiceFixture(t *testing.T) invoiceFixture {
	t.Helper()
	dsn := os.Getenv("PAYMENT_EMAIL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set PAYMENT_EMAIL_TEST_DATABASE_URL to a separate migrated Payment test database")
	}
	if dsn == os.Getenv("PAYMENT_TEST_DATABASE_URL") || dsn == os.Getenv("PAYMENT_OUTBOX_TEST_DATABASE_URL") {
		t.Fatal("email worker tests need a separate database")
	}
	t.Setenv("PAYMENT_TEST_DATABASE_URL", dsn)
	return newInvoiceFixture(t)
}

func emailJobs(t *testing.T, f invoiceFixture) []model.EmailDelivery {
	t.Helper()
	var jobs []model.EmailDelivery
	if err := f.db.Table("payment.email_deliveries").Where("booking_id = ?", f.event.AggregateID).Order("kind").Find(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	return jobs
}

func emailWorker(t *testing.T, f invoiceFixture, sender EmailSender) *EmailService {
	t.Helper()
	worker, err := NewEmailService(repository.NewPaymentRepository(f.db), sender, "http://localhost:5173", slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	worker.now = f.service.now
	return worker
}

func issueEmailInvoice(t *testing.T, f invoiceFixture) {
	t.Helper()
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); err != nil {
		t.Fatal(err)
	}
}

func TestEmailJobsAndSuccessfulDeliveryPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	jobs := emailJobs(t, f)
	if len(jobs) != 2 || jobs[0].Kind != "booking_confirmation" || jobs[1].Kind != "invoice" {
		t.Fatal("invoice did not queue two jobs")
	}
	for _, job := range jobs {
		if job.Status != "queued" || job.AttemptCount != 0 || job.Recipient != "customer@example.test" {
			t.Fatal("wrong recipient snapshot or initial state")
		}
	}
	recorder := &emailRecorder{}
	worker := emailWorker(t, f, recorder)
	if err := worker.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, job := range emailJobs(t, f) {
		if job.Status != "sent" || job.AttemptCount != 1 || job.SentAt == nil || job.NextAttemptAt != nil || job.LockedUntil != nil {
			t.Fatal("successful delivery not committed")
		}
	}
	issueEmailInvoice(t, f)
	restarted := emailWorker(t, f, recorder)
	if err := restarted.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(recorder.messages) != 2 || len(emailJobs(t, f)) != 2 {
		t.Fatal("restart/replay duplicated delivery")
	}
	for _, message := range recorder.messages {
		if !strings.Contains(message.Body, "WITA") || !strings.Contains(message.Body, "IDR 150000") || !strings.Contains(message.Body, "sign in required") || !strings.Contains(message.Body, "/account/bookings/"+f.event.AggregateID) || strings.Contains(message.Body, "va_number") {
			t.Fatal("email content did not use invoice snapshot")
		}
	}
}

func TestEmailRetryBudgetPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	recorder := &emailRecorder{err: errors.New("synthetic private SMTP response customer@example.test")}
	worker := emailWorker(t, f, recorder)
	var logs bytes.Buffer
	worker.log = slog.New(slog.NewJSONHandler(&logs, nil))
	now := worker.now()
	worker.now = func() time.Time { return now }
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour}
	for attempt := 1; attempt <= 5; attempt++ {
		if err := worker.ProcessBatch(t.Context()); err != nil {
			t.Fatal(err)
		}
		jobs := emailJobs(t, f)
		for _, job := range jobs {
			if job.AttemptCount != attempt || job.LastErrorCode == nil || *job.LastErrorCode != "SMTP_UNAVAILABLE" {
				t.Fatal("retry did not persist safe state")
			}
			if attempt < 5 {
				if job.Status != "queued" || job.NextAttemptAt == nil || !job.NextAttemptAt.Equal(now.Add(delays[attempt-1])) {
					t.Fatal("incorrect retry schedule")
				}
			} else if job.Status != "failed" || job.NextAttemptAt != nil {
				t.Fatal("retry budget not exhausted")
			}
		}
		before := len(recorder.messages)
		if err := worker.ProcessBatch(t.Context()); err != nil {
			t.Fatal(err)
		}
		if len(recorder.messages) != before {
			t.Fatal("retry ran before due or repeated terminal failure")
		}
		if attempt < 5 {
			now = now.Add(delays[attempt-1])
		}
	}
	if len(recorder.messages) != 10 || strings.Contains(logs.String(), "customer@example.test") || strings.Contains(logs.String(), "private SMTP") {
		t.Fatal("retry count or log privacy failed")
	}
	if countPaymentRows(t, f.db, "invoices", "booking_id", f.event.AggregateID) != 1 || reloadAttempt(t, f.db, f.event.Payload.PaymentID).Status != "paid" {
		t.Fatal("SMTP failure changed invoice/payment")
	}
}

func TestInvalidRecipientIsPermanentPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	recorder := &emailRecorder{err: mailer.ErrInvalidRecipient}
	worker := emailWorker(t, f, recorder)
	if err := worker.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, job := range emailJobs(t, f) {
		if job.Status != "failed" || job.AttemptCount != 1 || job.LastErrorCode == nil || *job.LastErrorCode != "INVALID_RECIPIENT" {
			t.Fatal("invalid recipient was retried")
		}
	}
}

func TestConcurrentEmailWorkersPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	recorder := &emailRecorder{}
	workers := []*EmailService{emailWorker(t, f, recorder), emailWorker(t, f, recorder)}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, worker := range workers {
		wg.Add(1)
		go func(w *EmailService) { defer wg.Done(); results <- w.ProcessBatch(t.Context()) }(worker)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(recorder.messages) != 2 || recorder.messages[0].ID == recorder.messages[1].ID {
		t.Fatal("two workers sent the same active job")
	}
}

func TestEmailClaimRecoveryAndStaleOwnerPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	repo := repository.NewPaymentRepository(f.db)
	now := f.service.now()
	old, claimed, err := repo.ClaimEmail(t.Context(), now)
	if err != nil || !claimed {
		t.Fatal("could not claim email")
	}
	if err := f.db.Exec("UPDATE payment.email_deliveries SET status = 'sent',sent_at = ?,next_attempt_at = NULL WHERE id <> ? AND booking_id = ?", now, old.ID, f.event.AggregateID).Error; err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Second)
	if err := repo.RecoverEmailClaims(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	current, claimed, err := repo.ClaimEmail(t.Context(), now)
	if err != nil || !claimed || current.ID != old.ID || current.MessageID != old.MessageID || current.AttemptCount != 2 {
		t.Fatal("restart did not preserve logical job")
	}
	if updated, err := repo.FinishEmail(t.Context(), old, "", now); err != nil || updated {
		t.Fatal("stale owner overwrote a new attempt")
	}
	if updated, err := repo.FinishEmail(t.Context(), current, "", now); err != nil || !updated {
		t.Fatal("current owner could not finish")
	}
	if err := f.db.Exec("UPDATE payment.email_deliveries SET status = 'sending',attempt_count = 5,locked_until = ? WHERE id = ?", now.Add(-time.Second), old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.RecoverEmailClaims(t.Context(), now); err != nil {
		t.Fatal(err)
	}
	for _, job := range emailJobs(t, f) {
		if job.ID == old.ID && (job.Status != "failed" || job.LastErrorCode == nil || *job.LastErrorCode != "DELIVERY_INTERRUPTED") {
			t.Fatal("exhausted interrupted job remained stuck")
		}
	}
}

func TestEmailInvoiceRollbackPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	if err := f.db.Exec(`CREATE FUNCTION payment.test_reject_email() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic job failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.db.Exec("DROP TRIGGER IF EXISTS test_reject_email ON payment.email_deliveries; DROP FUNCTION IF EXISTS payment.test_reject_email()").Error; err != nil {
			t.Error(err)
		}
	})
	if err := f.db.Exec("CREATE TRIGGER test_reject_email BEFORE INSERT ON payment.email_deliveries FOR EACH ROW EXECUTE FUNCTION payment.test_reject_email()").Error; err != nil {
		t.Fatal(err)
	}
	if err := f.service.Process(t.Context(), confirmedBody(t, f.event)); !errors.Is(err, repository.ErrUnavailable) {
		t.Fatal("job persistence failure not returned")
	}
	if len(emailJobs(t, f)) != 0 || countPaymentRows(t, f.db, "invoices", "booking_id", f.event.AggregateID) != 0 || countPaymentRows(t, f.db, "inbox_events", "aggregate_id", f.event.AggregateID) != 0 {
		t.Fatal("job failure left a partial invoice/inbox")
	}
	if err := f.db.Exec("DROP TRIGGER test_reject_email ON payment.email_deliveries").Error; err != nil {
		t.Fatal(err)
	}
	issueEmailInvoice(t, f)
	if len(emailJobs(t, f)) != 2 {
		t.Fatal("retry did not queue jobs")
	}
}

func TestEmailWorkerCancellationPostgres(t *testing.T) {
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	recorder := &emailRecorder{afterSend: cancel}
	worker := emailWorker(t, f, recorder)
	if err := worker.ProcessBatch(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("shutdown did not cancel the batch")
	}
	var originalID string
	for _, job := range emailJobs(t, f) {
		if job.Status == "sending" {
			originalID = job.MessageID
		}
	}
	if originalID == "" {
		t.Fatal("interrupted send did not retain its claim")
	}
	restarted := emailWorker(t, f, &emailRecorder{})
	restarted.now = func() time.Time { return f.service.now().Add(31 * time.Second) }
	if err := restarted.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, job := range emailJobs(t, f) {
		if job.Status != "sent" {
			t.Fatal("restart did not recover delivery")
		}
		if job.AttemptCount == 2 && job.MessageID != originalID {
			t.Fatal("retry changed Message-ID")
		}
	}
}

func TestEmailRunAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	worker := &EmailService{}
	worker.Run(ctx)
}
