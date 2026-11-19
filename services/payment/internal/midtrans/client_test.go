package midtrans

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateVA(t *testing.T) {
	deadline := time.Date(2026, 11, 20, 2, 15, 0, 123456000, time.UTC)
	for _, test := range []struct {
		name, change string
		wantError    error
	}{
		{"pending VA", "", nil},
		{"wrong amount", "amount", ErrInvalidResponse},
		{"fractional amount", "fraction", ErrInvalidResponse},
		{"wrong order", "order", ErrInvalidResponse},
		{"wrong currency", "currency", ErrInvalidResponse},
		{"wrong channel", "channel", ErrInvalidResponse},
		{"wrong bank", "bank", ErrInvalidResponse},
		{"missing VA", "va", ErrInvalidResponse},
		{"extended expiry", "extended", ErrInvalidResponse},
		{"missing expiry", "expiry", ErrInvalidResponse},
		{"unexpected settlement", "settlement", ErrInvalidResponse},
		{"malformed JSON", "json", ErrInvalidResponse},
		{"provider rejected", "rejected", ErrInvalidResponse},
		{"provider unavailable", "unavailable", ErrUnavailable},
		{"oversized response", "large", ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				if !ok || username != "test-only-key" || password != "" || r.URL.Path != "/v2/charge" || r.Method != "POST" || r.Header.Get("X-Request-ID") != "provider-test" {
					t.Error("incorrect provider authentication, path, or request ID")
				}
				var request struct {
					BankTransfer struct {
						Bank string `json:"bank"`
					} `json:"bank_transfer"`
					TransactionDetails struct {
						OrderID string `json:"order_id"`
						Amount  int64  `json:"gross_amount"`
					} `json:"transaction_details"`
					Expiry struct {
						OrderTime string `json:"order_time"`
						Duration  int    `json:"expiry_duration"`
						Unit      string `json:"unit"`
					} `json:"custom_expiry"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
				}
				start, err := time.Parse("2006-01-02 15:04:05 -0700", request.Expiry.OrderTime)
				if err != nil || !start.Add(time.Duration(request.Expiry.Duration)*time.Second).Equal(deadline.Truncate(time.Second)) || request.Expiry.Unit != "second" || request.BankTransfer.Bank != "bni" || request.TransactionDetails.Amount != 150000 {
					t.Error("provider request must preserve the original deadline and integer amount")
				}
				response := map[string]any{"order_id": "sroom-test", "transaction_id": "transaction-fixture", "payment_type": "bank_transfer",
					"gross_amount": "150000.00", "currency": "IDR", "transaction_status": "pending", "expiry_time": "2026-11-20 09:15:00",
					"va_numbers": []map[string]string{{"bank": "bni", "va_number": "0000000000000000"}}, "future_field": true}
				switch test.change {
				case "amount":
					response["gross_amount"] = "150001.00"
				case "fraction":
					response["gross_amount"] = "150000.01"
				case "order":
					response["order_id"] = "wrong"
				case "currency":
					response["currency"] = "USD"
				case "channel":
					response["payment_type"] = "credit_card"
				case "bank":
					response["va_numbers"] = []map[string]string{{"bank": "bca", "va_number": "0000"}}
				case "va":
					delete(response, "va_numbers")
				case "extended":
					response["expiry_time"] = "2026-11-20 09:15:01"
				case "expiry":
					delete(response, "expiry_time")
				case "settlement":
					response["transaction_status"] = "settlement"
				case "json":
					_, _ = w.Write([]byte("not JSON"))
					return
				case "rejected":
					w.WriteHeader(401)
				case "unavailable":
					w.WriteHeader(503)
				case "large":
					_, _ = w.Write([]byte(strings.Repeat("x", 64*1024+1)))
					return
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			client := NewClient("test-only-key")
			client.baseURL = server.URL
			result, err := client.CreateVA(t.Context(), ChargeInput{OrderID: "sroom-test", AmountIDR: 150000, HoldExpiresAt: deadline}, "provider-test")
			if !errors.Is(err, test.wantError) {
				t.Fatalf("error=%v, want %v", err, test.wantError)
			}
			if err == nil && (result.Bank != "bni" || result.VANumber == "" || result.ExpiresAt.Location() != time.UTC || len(result.Digest) != 64) {
				t.Fatal("invalid normalized provider result")
			}
		})
	}
}

func TestCreateVANetworkFailureAndRedirect(t *testing.T) {
	client := NewClient("test-only-key")
	client.baseURL = "http://127.0.0.1:1"
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.CreateVA(ctx, ChargeInput{}, "test"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("canceled request: %v", err)
	}
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	client.baseURL = redirect.URL
	_, err := client.CreateVA(t.Context(), ChargeInput{}, "test")
	if !errors.Is(err, ErrInvalidResponse) || called {
		t.Fatal("provider redirects must not forward credentials")
	}
}
