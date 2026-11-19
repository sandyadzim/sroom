package model

import "time"

type Invoice struct {
	ID              string
	InvoiceNumber   string
	BookingID       string
	PaymentID       string
	CustomerID      string
	ProviderOrderID string
	BilledToName    string
	BilledToEmail   string
	VariantID       string
	VehicleLabel    string
	PickupAt        time.Time
	ReturnAt        time.Time
	RentalDays      int
	DailyRateIDR    int64 `gorm:"column:daily_rate_idr"`
	TotalIDR        int64 `gorm:"column:total_idr"`
	Currency        string
	Bank            string
	ConfirmedAt     time.Time
	IssuedAt        time.Time
}
