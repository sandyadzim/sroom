package checkout

import (
	"testing"
	"time"
)

func TestReportPeriod(t *testing.T) {
	now := time.Date(2026, 11, 20, 16, 0, 0, 0, time.UTC) // Already November 21 in WITA.
	period, err := ParseReportPeriod("", "", now)
	if err != nil || period.From != "2026-11-15" || period.To != "2026-11-21" ||
		period.Start.UTC().Format(time.RFC3339) != "2026-11-14T16:00:00Z" || period.End.UTC().Format(time.RFC3339) != "2026-11-21T16:00:00Z" {
		t.Fatalf("incorrect WITA period: %+v, %v", period, err)
	}
	for _, input := range [][2]string{{"2026-02-30", "2026-03-01"}, {"2026-11-20", ""}, {"", "2026-11-20"}, {"2026-11-21", "2026-11-20"}, {"2024-01-01", "2025-01-01"}, {"0000-01-01", "0000-01-02"}} {
		if _, err := ParseReportPeriod(input[0], input[1], now); err != ErrInvalidReportPeriod {
			t.Errorf("accepted invalid period %v: %v", input, err)
		}
	}
	if _, err := ParseReportPeriod("2024-01-01", "2024-12-31", now); err != nil {
		t.Fatal("rejected 366-date leap year", err)
	}
}

func TestCompleteReport(t *testing.T) {
	period, _ := ParseReportPeriod("2026-11-18", "2026-11-20", time.Now())
	report := completeReport(Report{Period: period, Summary: ReportSummary{TotalBookings: 4, VerifiedConfirmedBookings: 1}, Daily: []ReportDay{{Date: "2026-11-19", Bookings: 4, ConfirmedValueIDR: 800000}}})
	if len(report.Daily) != 3 || report.Daily[0].Date != "2026-11-18" || report.Daily[0].Bookings != 0 || report.Daily[1].ConfirmedValueIDR != 800000 || report.Daily[2].Bookings != 0 || report.Summary.PaymentConversion == nil || *report.Summary.PaymentConversion != 0.25 {
		t.Fatalf("incorrect report: %+v", report)
	}
	empty := completeReport(Report{Period: period})
	if empty.Summary.PaymentConversion != nil || len(empty.Daily) != 3 || !empty.Simulated || empty.Timezone != "Asia/Makassar" || empty.Currency != "IDR" {
		t.Fatalf("incorrect empty report: %+v", empty)
	}
}
