package dto

import "time"

type ExpiryAssessment struct {
	BookingID           string     `json:"booking_id"`
	HoldExpiresAt       time.Time  `json:"hold_expires_at"`
	Result              string     `json:"result"`
	ClosedAt            *time.Time `json:"closed_at"`
	PaymentID           *string    `json:"payment_id"`
	ReleaseAuthorizedAt *time.Time `json:"release_authorized_at,omitempty"`
}
