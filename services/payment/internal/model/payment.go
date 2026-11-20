package model

import "time"

type PaymentAttempt struct {
	ID                    string
	Provider              string
	BookingID             string
	CustomerID            string
	CustomerFullName      *string
	CustomerEmail         *string
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
	VerifiedAt            *time.Time
	ProviderPaidAt        *time.Time
	ReleaseAuthorizedAt   *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ProviderObservation struct {
	ID                    string
	ObservedAt            time.Time
	PaymentID             string
	Source                string
	ProviderTransactionID *string
	ProviderStatus        *string
	AmountIDR             *int64 `gorm:"column:amount_idr"`
	Currency              *string
	VerificationResult    string
	PayloadDigest         *string
	ErrorCode             *string
	RequestID             string
}

type WebhookReceipt struct {
	ID              string
	PaymentID       *string
	ProviderOrderID string
	PayloadHash     string
	Status          string
}

type PaymentOutcome struct {
	Status            string
	ProviderStatus    string
	TransactionID     string
	ReviewReason      *string
	Observation       ProviderObservation
	VerifiedAt        time.Time
	VANumber          *string
	ProviderExpiresAt *time.Time
	AuthorizeRelease  bool
}
