package midtrans

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func VerifySignature(orderID, statusCode, grossAmount, signature, key string) bool {
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != sha512.Size || key == "" {
		return false
	}
	expected := sha512.Sum512([]byte(orderID + statusCode + grossAmount + key))
	return subtle.ConstantTimeCompare(provided, expected[:]) == 1
}

func ParseAmount(value string) (int64, error) {
	whole, fraction, _ := strings.Cut(value, ".")
	amount, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || amount <= 0 || strings.Trim(fraction, "0") != "" {
		return 0, ErrInvalidResponse
	}
	return amount, nil
}

type StatusResult struct {
	OrderID, TransactionID, MerchantID, Status, StatusCode string
	PaymentType, Currency, Bank, FraudStatus               string
	AmountIDR                                              int64
	Digest                                                 string
}

func (c *Client) GetStatus(ctx context.Context, orderID, requestID string) (StatusResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/"+url.PathEscape(orderID)+"/status", nil)
	if err != nil {
		return StatusResult{}, ErrUnavailable
	}
	req.SetBasicAuth(c.key, "")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Request-ID", requestID)
	res, err := c.http.Do(req)
	if err != nil {
		return StatusResult{}, ErrUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024+1))
	if err != nil || len(body) > 64*1024 {
		return StatusResult{}, ErrUnavailable
	}
	digest := sha256.Sum256(body)
	result := StatusResult{Digest: hex.EncodeToString(digest[:])}
	if res.StatusCode != http.StatusOK {
		return result, ErrUnavailable
	}
	var response struct {
		OrderID       string `json:"order_id"`
		TransactionID string `json:"transaction_id"`
		MerchantID    string `json:"merchant_id"`
		Status        string `json:"transaction_status"`
		StatusCode    string `json:"status_code"`
		PaymentType   string `json:"payment_type"`
		Currency      string `json:"currency"`
		GrossAmount   string `json:"gross_amount"`
		FraudStatus   string `json:"fraud_status"`
		Bank          string `json:"bank"`
		VANumbers     []struct {
			Bank string `json:"bank"`
		} `json:"va_numbers"`
	}
	if json.Unmarshal(body, &response) != nil {
		return result, ErrInvalidResponse
	}
	result.OrderID, result.TransactionID, result.MerchantID = response.OrderID, response.TransactionID, response.MerchantID
	result.Status, result.StatusCode = response.Status, response.StatusCode
	result.PaymentType, result.Currency, result.FraudStatus = response.PaymentType, response.Currency, response.FraudStatus
	result.Bank = response.Bank
	if result.Bank == "" && len(response.VANumbers) == 1 {
		result.Bank = response.VANumbers[0].Bank
	}
	result.AmountIDR, err = ParseAmount(response.GrossAmount)
	if err != nil || result.OrderID == "" || result.TransactionID == "" || result.MerchantID == "" || result.Status == "" || result.Currency == "" || result.PaymentType == "" {
		return result, ErrInvalidResponse
	}
	return result, nil
}
