package dto

import "time"

type PaymentConfirmation struct {
	PaymentID       string     `json:"payment_id"`
	BookingID       string     `json:"booking_id"`
	CustomerID      string     `json:"customer_id"`
	ProviderOrderID string     `json:"provider_order_id"`
	AmountIDR       int64      `json:"amount_idr"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	HoldExpiresAt   time.Time  `json:"hold_expires_at"`
	VerifiedAt      *time.Time `json:"verified_at"`
	ProviderPaidAt  *time.Time `json:"provider_paid_at"`
	ReviewRequired  bool       `json:"review_required"`
}
