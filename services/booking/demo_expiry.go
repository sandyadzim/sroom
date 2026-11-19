package main

import (
	"context"
	"log/slog"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
)

func startDemoExpiry(ctx context.Context, service *checkout.Service, cfg config, log *slog.Logger) func() {
	if !cfg.demoPayments || cfg.paymentToken == "" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runDemoExpiry(ctx, func(ctx context.Context) {
			requestID := uuid.NewString()
			result, err := service.ExpireDemoHolds(ctx, time.Now, requestID)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				log.Warn("demo expiry incomplete", "operation", "bookings.demo_expire", "request_id", requestID,
					"expired", result.Expired, "failed", result.Failed)
			} else if result.Expired > 0 {
				log.Info("demo holds expired", "operation", "bookings.demo_expire", "request_id", requestID,
					"expired", result.Expired)
			}
		})
	}()
	return func() {
		cancel()
		<-done
	}
}

// The batch function keeps database work separate from the polling schedule.
func runDemoExpiry(ctx context.Context, expire func(context.Context)) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		expire(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
