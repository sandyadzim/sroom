package midtrans

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const SandboxURL = "https://api.sandbox.midtrans.com"

var (
	ErrUnavailable     = errors.New("payment provider is unavailable")
	ErrInvalidResponse = errors.New("payment provider returned an unexpected response")
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

// The runtime always uses SandboxURL. Tests can inject a local HTTP server.
func NewClient(key string) *Client {
	return &Client{baseURL: SandboxURL, key: key, http: &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type ChargeInput struct {
	OrderID       string
	AmountIDR     int64
	HoldExpiresAt time.Time
}

type ChargeResult struct {
	OrderID, TransactionID, Status, Currency, Bank, VANumber string
	AmountIDR                                                int64
	ExpiresAt                                                time.Time
	Digest                                                   string
}

func (c *Client) CreateVA(ctx context.Context, input ChargeInput, requestID string) (ChargeResult, error) {
	wib := time.FixedZone("WIB", 7*60*60)
	deadline := input.HoldExpiresAt.UTC().Truncate(time.Second)
	var payload struct {
		PaymentType        string `json:"payment_type"`
		TransactionDetails struct {
			OrderID     string `json:"order_id"`
			GrossAmount int64  `json:"gross_amount"`
		} `json:"transaction_details"`
		BankTransfer struct {
			Bank string `json:"bank"`
		} `json:"bank_transfer"`
		CustomExpiry struct {
			OrderTime string `json:"order_time"`
			Duration  int    `json:"expiry_duration"`
			Unit      string `json:"unit"`
		} `json:"custom_expiry"`
	}
	payload.PaymentType = "bank_transfer"
	payload.TransactionDetails.OrderID, payload.TransactionDetails.GrossAmount = input.OrderID, input.AmountIDR
	payload.BankTransfer.Bank = "bni"
	payload.CustomExpiry.OrderTime = deadline.Add(-15 * time.Minute).In(wib).Format("2006-01-02 15:04:05 -0700")
	payload.CustomExpiry.Duration, payload.CustomExpiry.Unit = 900, "second"
	body, err := json.Marshal(payload)
	if err != nil {
		return ChargeResult{}, ErrInvalidResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/charge", bytes.NewReader(body))
	if err != nil {
		return ChargeResult{}, ErrUnavailable
	}
	req.SetBasicAuth(c.key, "")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", requestID)
	res, err := c.http.Do(req)
	if err != nil {
		return ChargeResult{}, ErrUnavailable
	}
	defer res.Body.Close()
	body, err = io.ReadAll(io.LimitReader(res.Body, 64*1024+1))
	if err != nil || len(body) > 64*1024 {
		return ChargeResult{}, ErrUnavailable
	}
	digest := sha256.Sum256(body)
	result := ChargeResult{Digest: hex.EncodeToString(digest[:])}
	if res.StatusCode >= 500 {
		return result, ErrUnavailable
	}
	var response struct {
		OrderID       string `json:"order_id"`
		TransactionID string `json:"transaction_id"`
		PaymentType   string `json:"payment_type"`
		Status        string `json:"transaction_status"`
		Currency      string `json:"currency"`
		GrossAmount   string `json:"gross_amount"`
		ExpiryTime    string `json:"expiry_time"`
		VANumbers     []struct {
			Bank   string `json:"bank"`
			Number string `json:"va_number"`
		} `json:"va_numbers"`
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 || json.Unmarshal(body, &response) != nil {
		return result, ErrInvalidResponse
	}
	result.OrderID, result.TransactionID, result.Status, result.Currency = response.OrderID, response.TransactionID, response.Status, response.Currency
	amount, fraction, _ := strings.Cut(response.GrossAmount, ".")
	result.AmountIDR, err = strconv.ParseInt(amount, 10, 64)
	if err != nil || strings.Trim(fraction, "0") != "" || result.AmountIDR <= 0 {
		return result, ErrInvalidResponse
	}
	result.ExpiresAt, err = time.ParseInLocation("2006-01-02 15:04:05", response.ExpiryTime, wib)
	if err != nil || response.PaymentType != "bank_transfer" || response.TransactionID == "" ||
		result.OrderID != input.OrderID || result.AmountIDR != input.AmountIDR || result.Currency != "IDR" ||
		result.Status != "pending" || result.ExpiresAt.After(deadline) {
		return result, ErrInvalidResponse
	}
	for _, va := range response.VANumbers {
		if va.Bank == "bni" && va.Number != "" && strings.Trim(va.Number, "0123456789") == "" {
			result.Bank, result.VANumber = va.Bank, va.Number
			result.ExpiresAt = result.ExpiresAt.UTC()
			return result, nil
		}
	}
	return result, ErrInvalidResponse
}
