package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("payment confirmation unavailable")
var ErrNotFound = errors.New("payment not found")

type Confirmation struct {
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

type Client struct {
	baseURL, token string
	http           *http.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" || base.User != nil ||
		base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return nil, errors.New("PAYMENT_BASE_URL must be an HTTP origin")
	}
	return &Client{baseURL: strings.TrimRight(base.String(), "/"), token: token, http: &http.Client{Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Confirmation(ctx context.Context, bookingID, requestID string) (Confirmation, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/internal/v1/bookings/"+bookingID+"/payment-confirmation", nil)
	if err != nil {
		return Confirmation{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Request-ID", requestID)
	response, err := c.http.Do(req)
	if err != nil {
		return Confirmation{}, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return Confirmation{}, ErrNotFound
	}
	if response.StatusCode != 200 {
		return Confirmation{}, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024+1))
	if err != nil || len(body) > 16*1024 {
		return Confirmation{}, ErrUnavailable
	}
	var result struct {
		Data Confirmation `json:"data"`
	}
	if json.Unmarshal(body, &result) != nil {
		return Confirmation{}, ErrUnavailable
	}
	data := result.Data
	for _, id := range []string{data.PaymentID, data.BookingID, data.CustomerID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return Confirmation{}, ErrUnavailable
		}
	}
	if data.BookingID != bookingID || data.AmountIDR <= 0 || data.Currency != "IDR" || data.ProviderOrderID == "" ||
		len(data.ProviderOrderID) > 50 || data.HoldExpiresAt.IsZero() || data.Status == "" ||
		(data.Status == "paid" && (data.VerifiedAt == nil || data.VerifiedAt.IsZero())) {
		return Confirmation{}, ErrUnavailable
	}
	return data, nil
}
