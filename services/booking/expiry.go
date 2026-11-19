package main

import (
	"context"
	"log/slog"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
)

func startPaymentExpiry(ctx context.Context, service *checkout.ExpiryService, cfg config, log *slog.Logger) func() {
	if cfg.rabbitURL == "" || cfg.paymentToken == "" || cfg.demoPayments {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runExpiryChecks(ctx, func(ctx context.Context) {
			requestID := uuid.NewString()
			result, err := service.ExpireHolds(ctx, time.Now, requestID)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Warn("hold expiry incomplete", "operation", "bookings.expire", "request_id", requestID,
					"expired", result.Expired, "held", result.Held, "failed", result.Failed)
			} else if result.Expired > 0 || result.Held > 0 {
				log.Info("hold expiry checked", "operation", "bookings.expire", "request_id", requestID,
					"expired", result.Expired, "held", result.Held)
			}
		})
	}()
	return func() {
		cancel()
		<-done
	}
}
