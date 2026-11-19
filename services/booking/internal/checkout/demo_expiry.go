package checkout

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidDemoAllocation = errors.New("booking allocation is not eligible for demo expiry")

type DemoExpiryResult struct {
	Expired int
	Failed  int
}

// ExpireDemoHolds assumes unpaid demo holds. Real payments need provider reconciliation.
func (s *Service) ExpireDemoHolds(ctx context.Context, now func() time.Time, requestID string) (DemoExpiryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var result DemoExpiryResult
	candidates, err := s.allocations.DueHolds(ctx, now())
	if err != nil {
		return result, err
	}
	var batchError error
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		expired, err := s.allocations.expireDemoHold(ctx, candidate.BookingID, now, requestID)
		if err != nil {
			result.Failed++
			batchError = err
			continue
		}
		if expired {
			result.Expired++
		}
	}
	return result, batchError
}
