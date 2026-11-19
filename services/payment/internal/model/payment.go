package model

import "time"

type PaymentAttempt struct {
	ID                    string
	BookingID             string
	CustomerID            string
	ProviderOrderID       string
	ProviderTransactionID *string
	AmountIDR             int64 `gorm:"column:amount_idr"`
	Currency              string
	Bank                  string
	VANumber              *string
	Status                string
	ProviderStatus        *string
	HoldExpiresAt         time.Time
	ProviderExpiresAt     *time.Time
	ReviewReason          *string
}

type ProviderObservation struct {
	PaymentID             string
	ProviderTransactionID *string
	ProviderStatus        *string
	AmountIDR             *int64 `gorm:"column:amount_idr"`
	Currency              *string
	VerificationResult    string
	PayloadDigest         *string
	ErrorCode             *string
	RequestID             string
}
