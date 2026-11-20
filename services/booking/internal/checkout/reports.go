package checkout

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidReportPeriod = errors.New("send from and to as YYYY-MM-DD, in order, covering at most 366 dates")

type ReportPeriod struct {
	From  string    `json:"from"`
	To    string    `json:"to"`
	Start time.Time `json:"-"`
	End   time.Time `json:"-"`
}

type ReportSummary struct {
	TotalBookings             int64    `json:"total_bookings"`
	PendingPaymentBookings    int64    `json:"pending_payment_bookings"`
	ConfirmedBookings         int64    `json:"confirmed_bookings"`
	ExpiredBookings           int64    `json:"expired_bookings"`
	PaymentReviewBookings     int64    `json:"payment_review_bookings"`
	VerifiedConfirmedBookings int64    `json:"verified_confirmed_bookings"`
	ConfirmedValueIDR         int64    `json:"confirmed_value_idr" gorm:"column:confirmed_value_idr"`
	PaymentConversion         *float64 `json:"payment_conversion" gorm:"-"`
}

type ReportDay struct {
	Date              string `json:"date"`
	Bookings          int64  `json:"bookings"`
	ConfirmedValueIDR int64  `json:"confirmed_value_idr" gorm:"column:confirmed_value_idr"`
}

type ReportVariant struct {
	VariantID    string `json:"variant_id"`
	VehicleLabel string `json:"vehicle_label"`
	Bookings     int64  `json:"bookings"`
}

type Report struct {
	Period    ReportPeriod  `json:"period"`
	Timezone  string        `json:"timezone"`
	Currency  string        `json:"currency"`
	Simulated bool          `json:"simulated"`
	Summary   ReportSummary `json:"summary"`
	Daily     []ReportDay   `json:"daily"`
}

type Reports struct {
	Report
	TopVariants []ReportVariant `json:"top_variants"`
}

type Overview struct {
	Report
	RecentBookings []StaffBooking `json:"recent_bookings"`
}

func ParseReportPeriod(from, to string, now time.Time) (ReportPeriod, error) {
	wita, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		return ReportPeriod{}, err
	}
	if from == "" && to == "" {
		today := now.In(wita)
		to = today.Format(time.DateOnly)
		from = today.AddDate(0, 0, -6).Format(time.DateOnly)
	}
	start, startErr := time.ParseInLocation(time.DateOnly, from, wita)
	last, lastErr := time.ParseInLocation(time.DateOnly, to, wita)
	if startErr != nil || lastErr != nil || start.Year() < 1 || last.Year() > 9999 || last.Before(start) || last.After(start.AddDate(0, 0, 365)) {
		return ReportPeriod{}, ErrInvalidReportPeriod
	}
	return ReportPeriod{From: from, To: to, Start: start, End: last.AddDate(0, 0, 1)}, nil
}

func (s *Service) Reports(ctx context.Context, period ReportPeriod) (Reports, error) {
	report, variants, _, err := s.allocations.readReport(ctx, period, false)
	return Reports{Report: completeReport(report), TopVariants: variants}, err
}

func (s *Service) Overview(ctx context.Context, period ReportPeriod) (Overview, error) {
	report, _, recent, err := s.allocations.readReport(ctx, period, true)
	for i := range recent {
		recent[i].Booking = s.localBooking(recent[i].Booking)
	}
	return Overview{Report: completeReport(report), RecentBookings: recent}, err
}

func completeReport(report Report) Report {
	report.Timezone = "Asia/Makassar"
	report.Currency = "IDR"
	report.Simulated = true
	if report.Summary.TotalBookings > 0 {
		conversion := float64(report.Summary.VerifiedConfirmedBookings) / float64(report.Summary.TotalBookings)
		report.Summary.PaymentConversion = &conversion
	}
	// SQL returns dates that contain bookings. Fill missing dates for a continuous chart.
	byDate := make(map[string]ReportDay, len(report.Daily))
	for _, day := range report.Daily {
		byDate[day.Date] = day
	}
	report.Daily = []ReportDay{}
	for date := report.Period.Start; date.Before(report.Period.End); date = date.AddDate(0, 0, 1) {
		label := date.Format(time.DateOnly)
		day := byDate[label]
		day.Date = label
		report.Daily = append(report.Daily, day)
	}
	return report
}
