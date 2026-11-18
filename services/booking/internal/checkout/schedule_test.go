package checkout

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestRentalDays(t *testing.T) {
	wita, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, now, pickup, returnAt string
		wantDays                    int
	}{
		{"three inclusive days", "2026-11-09T15:59:59Z", "2026-11-10T08:00:00+08:00", "2026-11-12T18:00:00+08:00", 3},
		{"same day is one day", "2026-11-09T15:59:59Z", "2026-11-10T00:00:00Z", "2026-11-10T00:30:00Z", 1},
		{"today after WITA midnight", "2026-11-09T16:00:00Z", "2026-11-10T08:00:00+08:00", "2026-11-11T18:00:00+08:00", 0},
		{"early slot", "2026-11-09T00:00:00Z", "2026-11-10T07:30:00+08:00", "2026-11-12T18:00:00+08:00", 0},
		{"late slot", "2026-11-09T00:00:00Z", "2026-11-10T08:30:00+08:00", "2026-11-12T18:30:00+08:00", 0},
		{"off-slot minute", "2026-11-09T00:00:00Z", "2026-11-10T09:15:00+08:00", "2026-11-12T18:00:00+08:00", 0},
		{"off-slot second", "2026-11-09T00:00:00Z", "2026-11-10T09:00:01+08:00", "2026-11-12T18:00:00+08:00", 0},
		{"return at pickup", "2026-11-09T00:00:00Z", "2026-11-10T09:00:00+08:00", "2026-11-10T09:00:00+08:00", 0},
		{"return before pickup", "2026-11-09T00:00:00Z", "2026-11-10T09:00:00+08:00", "2026-11-10T08:30:00+08:00", 0},
		{"more than 30 days", "2026-11-09T00:00:00Z", "2026-11-10T09:00:00+08:00", "2026-12-10T08:00:00+08:00", 31},
	} {
		t.Run(test.name, func(t *testing.T) {
			local := func(value string) time.Time {
				parsed, err := time.Parse(time.RFC3339, value)
				if err != nil {
					t.Fatal(err)
				}
				return parsed.In(wita)
			}
			days, err := rentalDays(local(test.pickup), local(test.returnAt), local(test.now))
			if test.wantDays == 0 {
				if !errors.Is(err, ErrInvalidSchedule) {
					t.Fatalf("error = %v, want invalid schedule", err)
				}
				return
			}
			if err != nil || days != test.wantDays {
				t.Fatalf("rental days = %d, error = %v, want %d", days, err, test.wantDays)
			}
		})
	}
}

func TestQuoteTotal(t *testing.T) {
	if total, err := quoteTotal(450000, 3); err != nil || total != 1350000 {
		t.Fatalf("total = %d, error = %v", total, err)
	}
	if _, err := quoteTotal(math.MaxInt64, 2); !errors.Is(err, ErrTotalOutOfRange) {
		t.Fatalf("error = %v, want overflow", err)
	}
}
