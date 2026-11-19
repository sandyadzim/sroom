package dto

import "time"

type ListPaymentsQuery struct {
	Page           int    `form:"page" binding:"min=1"`
	Limit          int    `form:"limit" binding:"min=1,max=100"`
	Search         string `form:"search" binding:"max=200"`
	Status         string `form:"status" binding:"omitempty,oneof=initiating pending paid expired failed"`
	Reconciliation string `form:"reconciliation" binding:"omitempty,oneof=required clear"`
}

type PaymentPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

type CustomerPaymentResponse struct {
	PaymentID         *string    `json:"payment_id"`
	BookingID         string     `json:"booking_id"`
	BookingStatus     string     `json:"booking_status"`
	Status            string     `json:"status"`
	AmountIDR         int64      `json:"amount_idr"`
	Currency          string     `json:"currency"`
	Bank              *string    `json:"bank"`
	VANumber          *string    `json:"va_number"`
	HoldExpiresAt     time.Time  `json:"hold_expires_at"`
	ProviderExpiresAt *time.Time `json:"provider_expires_at"`
	VerifiedAt        *time.Time `json:"verified_at"`
	ProviderPaidAt    *time.Time `json:"provider_paid_at"`
	ReviewRequired    bool       `json:"review_required"`
}

type StaffPaymentResponse struct {
	PaymentID             string    `json:"payment_id"`
	BookingID             string    `json:"booking_id"`
	CustomerID            string    `json:"customer_id"`
	Provider              string    `json:"provider"`
	ProviderOrderID       string    `json:"provider_order_id"`
	ProviderTransactionID *string   `json:"provider_transaction_id"`
	Status                string    `json:"status"`
	Bank                  string    `json:"bank"`
	AmountIDR             int64     `json:"amount_idr"`
	Currency              string    `json:"currency"`
	ReviewRequired        bool      `json:"review_required"`
	ReviewReason          *string   `json:"review_reason"`
	HoldExpiresAt         time.Time `json:"hold_expires_at"`
	ExpiryReviewRequired  bool      `json:"expiry_review_required"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type StaffPaymentDetail struct {
	StaffPaymentResponse
	BookingStatus     string                 `json:"booking_status"`
	ProviderStatus    *string                `json:"provider_status"`
	ProviderExpiresAt *time.Time             `json:"provider_expires_at"`
	VerifiedAt        *time.Time             `json:"verified_at"`
	ProviderPaidAt    *time.Time             `json:"provider_paid_at"`
	Timeline          []PaymentTimelineEntry `json:"timeline"`
	EmailDeliveries   []EmailDeliveryStatus  `json:"email_deliveries"`
}

type PaymentTimelineEntry struct {
	ID                 string    `json:"id"`
	Source             string    `json:"source"`
	OccurredAt         time.Time `json:"occurred_at"`
	ProviderStatus     *string   `json:"provider_status"`
	VerificationResult *string   `json:"verification_result"`
	ErrorCode          *string   `json:"error_code"`
}
