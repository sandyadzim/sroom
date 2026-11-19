package dto

import "time"

type InvoiceCustomer struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type InvoiceResponse struct {
	ID              string          `json:"id"`
	InvoiceNumber   string          `json:"invoice_number"`
	BookingID       string          `json:"booking_id"`
	PaymentID       string          `json:"payment_id"`
	CustomerID      string          `json:"customer_id"`
	ProviderOrderID string          `json:"provider_order_id"`
	BilledTo        InvoiceCustomer `json:"billed_to"`
	VariantID       string          `json:"variant_id"`
	VehicleLabel    string          `json:"vehicle_label"`
	PickupAt        time.Time       `json:"pickup_at"`
	ReturnAt        time.Time       `json:"return_at"`
	RentalDays      int             `json:"rental_days"`
	DailyRateIDR    int64           `json:"daily_rate_idr"`
	TotalIDR        int64           `json:"total_idr"`
	Currency        string          `json:"currency"`
	Bank            string          `json:"bank"`
	PaymentMethod   string          `json:"payment_method"`
	Status          string          `json:"status"`
	DocumentType    string          `json:"document_type"`
	ConfirmedAt     time.Time       `json:"confirmed_at"`
	IssuedAt        time.Time       `json:"issued_at"`
}
