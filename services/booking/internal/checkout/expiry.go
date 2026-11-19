package checkout

import (
	"context"
	"errors"
	"time"

	"booking/internal/payment"
)

type ExpiryReader interface {
	AssessExpiry(context.Context, string, string) (payment.ExpiryAssessment, error)
}

type ExpiryService struct {
	repo     *Repository
	payments ExpiryReader
	cursor   HoldCursor
}

type ExpiryResult struct {
	Expired int
	Held    int
	Failed  int
}

func NewExpiryService(repo *Repository, payments ExpiryReader) *ExpiryService {
	return &ExpiryService{repo: repo, payments: payments}
}

// One worker owns the cursor. Restart safely starts the scan from the beginning.
func (s *ExpiryService) ExpireHolds(ctx context.Context, now func() time.Time, requestID string) (ExpiryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var result ExpiryResult
	candidates, err := s.repo.dueHoldsAfter(ctx, now(), s.cursor)
	if err != nil {
		return result, err
	}
	if len(candidates) == 0 {
		s.cursor = HoldCursor{}
		return result, nil
	}
	var batchError error
	for _, candidate := range candidates {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		assessment, err := s.payments.AssessExpiry(ctx, candidate.BookingID, requestID)
		if err == nil && (assessment.BookingID != candidate.BookingID || !assessment.HoldExpiresAt.Equal(candidate.HoldExpiresAt)) {
			err = ErrInvalidExpiryAssessment
		}
		if err == nil {
			if assessment.Result == "no_attempt_closed" {
				var expired bool
				expired, err = s.repo.expireUnusedHold(ctx, assessment, now, requestID)
				if expired {
					result.Expired++
				}
			} else if assessment.Result == "attempt_exists" {
				result.Held++
			} else {
				err = ErrInvalidExpiryAssessment
			}
		}
		if ctx.Err() != nil {
			// Retry this candidate on the next tick if cancellation interrupted it.
			return result, ctx.Err()
		}
		s.cursor = HoldCursor{Deadline: candidate.HoldExpiresAt, ID: candidate.BookingID}
		if err != nil && !errors.Is(err, payment.ErrNotExpirable) {
			result.Failed++
			batchError = err
		}
	}
	if len(candidates) < 50 {
		s.cursor = HoldCursor{}
	}
	return result, batchError
}
