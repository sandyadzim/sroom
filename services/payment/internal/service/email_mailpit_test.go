package service

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"payment/internal/mailer"
	"payment/internal/repository"
)

// Opt-in: use a temporary local Mailpit, never an application/VPS inbox.
func TestMailpitDeliveryPostgres(t *testing.T) {
	address, base := os.Getenv("PAYMENT_MAILPIT_SMTP_ADDR"), os.Getenv("PAYMENT_MAILPIT_HTTP_URL")
	if address == "" || base == "" {
		t.Skip("set test Mailpit SMTP/HTTP addresses for the opt-in local smoke test")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil || (host != "127.0.0.1" && host != "localhost") {
		t.Fatal("Mailpit smoke test requires loopback SMTP")
	}
	web, err := url.Parse(base)
	if err != nil || web.Scheme != "http" || (web.Hostname() != "127.0.0.1" && web.Hostname() != "localhost") || web.User != nil {
		t.Fatal("Mailpit smoke test requires loopback HTTP")
	}
	f := emailInvoiceFixture(t)
	issueEmailInvoice(t, f)
	sender, err := mailer.NewSMTP(address, "SROOM <noreply@sroom.test>")
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewEmailService(repository.NewPaymentRepository(f.db), sender, "http://localhost:5173", slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	worker.now = f.service.now
	if err := worker.ProcessBatch(t.Context()); err != nil {
		t.Fatal(err)
	}
	wantIDs := map[string]bool{}
	for _, job := range emailJobs(t, f) {
		if job.Status != "sent" || job.AttemptCount != 1 {
			t.Fatal("Mailpit did not accept both emails")
		}
		wantIDs[strings.Trim(job.MessageID, "<>")] = false
	}
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(strings.TrimRight(base, "/") + "/api/v1/messages")
	if err != nil {
		t.Fatal("could not read test Mailpit")
	}
	var list struct {
		Messages []struct {
			ID        string
			MessageID string
		}
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&list)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		t.Fatal("invalid Mailpit message list")
	}
	for _, summary := range list.Messages {
		key := strings.Trim(summary.MessageID, "<>")
		if _, wanted := wantIDs[key]; !wanted {
			continue
		}
		response, err := client.Get(strings.TrimRight(base, "/") + "/api/v1/message/" + summary.ID)
		if err != nil {
			t.Fatal("could not read Mailpit test message")
		}
		var message struct {
			Subject string
			Text    string
			HTML    string
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&message)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || !strings.Contains(message.Text, f.event.AggregateID) || !strings.Contains(message.Text, "WITA") || !strings.Contains(message.Text, "sign in required") || message.HTML != "" {
			t.Fatal("Mailpit content did not match text template")
		}
		if !strings.Contains(message.Text, "http://localhost:5173/account/bookings/"+f.event.AggregateID) {
			t.Fatal("missing private booking link")
		}
		wantIDs[key] = true
	}
	for _, found := range wantIDs {
		if !found {
			t.Fatal("sent Message-ID was missing from Mailpit")
		}
	}
	t.Log("Mailpit accepted both synthetic text emails; stable Message-IDs, WITA and private links verified")
}
