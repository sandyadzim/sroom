package dto

// Keep the signed fields as strings exactly as Midtrans sent them.
type MidtransNotification struct {
	OrderID       string `json:"order_id"`
	StatusCode    string `json:"status_code"`
	GrossAmount   string `json:"gross_amount"`
	SignatureKey  string `json:"signature_key"`
	TransactionID string `json:"transaction_id"`
	MerchantID    string `json:"merchant_id"`
	PaymentType   string `json:"payment_type"`
	Currency      string `json:"currency"`
	Status        string `json:"transaction_status"`
}
