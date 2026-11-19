package dto

import "time"

type CreateVARequest struct {
	BookingID      string `json:"-"`
	CustomerID     string `json:"-"`
	IdempotencyKey string `json:"-"`
}

type VAResponse struct {
	PaymentID         string     `json:"payment_id"`
	BookingID         string     `json:"booking_id"`
	Status            string     `json:"status"`
	AmountIDR         int64      `json:"amount_idr"`
	Currency          string     `json:"currency"`
	Bank              string     `json:"bank"`
	VANumber          *string    `json:"va_number"`
	HoldExpiresAt     time.Time  `json:"hold_expires_at"`
	ProviderExpiresAt *time.Time `json:"provider_expires_at"`
	ReviewRequired    bool       `json:"review_required"`
}
