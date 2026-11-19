package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"payment/internal/mailer"
	"payment/internal/repository"
)

type EmailSender interface {
	Send(context.Context, mailer.Message) error
}

type EmailService struct {
	repo       *repository.PaymentRepository
	sender     EmailSender
	webBaseURL string
	wita       *time.Location
	log        *slog.Logger
	now        func() time.Time
}

func NewEmailService(repo *repository.PaymentRepository, sender EmailSender, webBaseURL string, log *slog.Logger) (*EmailService, error) {
	wita, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		return nil, errors.New("could not load Asia/Makassar time zone")
	}
	return &EmailService{repo: repo, sender: sender, webBaseURL: strings.TrimRight(webBaseURL, "/"), wita: wita, log: log, now: time.Now}, nil
}

func (s *EmailService) Run(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if err := s.ProcessBatch(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("email worker unavailable", "operation", "email.batch")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *EmailService) ProcessBatch(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := s.repo.RecoverEmailClaims(ctx, s.now().UTC()); err != nil {
		return err
	}
	for range 10 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		job, claimed, err := s.repo.ClaimEmail(ctx, s.now().UTC())
		if err != nil {
			return err
		}
		if !claimed {
			return nil
		}
		invoice, err := s.repo.FindInvoice(ctx, job.BookingID)
		if err != nil {
			return err
		}
		message, err := s.message(job, invoice)
		if err == nil {
			err = s.sender.Send(ctx, message)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		code := ""
		switch {
		case errors.Is(err, mailer.ErrInvalidRecipient):
			code = "INVALID_RECIPIENT"
		case errors.Is(err, mailer.ErrInvalidMessage):
			code = "INVALID_MESSAGE"
		case err != nil:
			code = "SMTP_UNAVAILABLE"
		}
		updated, err := s.repo.FinishEmail(ctx, job, code, s.now().UTC())
		if err != nil {
			return err
		}
		if !updated {
			s.log.Warn("email claim changed", "operation", "email.finish", "delivery_id", job.ID)
			continue
		}
		if code != "" {
			s.log.Warn("email delivery incomplete", "operation", "email.send", "delivery_id", job.ID, "kind", job.Kind, "error_code", code)
		}
	}
	return nil
}
