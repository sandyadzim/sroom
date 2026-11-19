package booking

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("booking not found")
	ErrUnavailable = errors.New("booking is unavailable")
)

type PaymentContext struct {
	BookingID     string    `json:"booking_id"`
	CustomerID    string    `json:"customer_id"`
	TotalIDR      int64     `json:"total_idr"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	HoldExpiresAt time.Time `json:"hold_expires_at"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" ||
		(base.Path != "" && base.Path != "/") {
		return nil, errors.New("BOOKING_BASE_URL must be an HTTP origin")
	}
	return &Client{baseURL: strings.TrimRight(base.String(), "/"), token: token, http: &http.Client{
		Timeout:       2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *Client) PaymentContext(ctx context.Context, bookingID, requestID string) (PaymentContext, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/v1/bookings/"+bookingID+"/payment-context", nil)
	if err != nil {
		return PaymentContext{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Request-ID", requestID)
	res, err := c.http.Do(req)
	if err != nil {
		return PaymentContext{}, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return PaymentContext{}, ErrNotFound
	}
	if res.StatusCode != http.StatusOK {
		return PaymentContext{}, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 16*1024+1))
	if err != nil || len(body) > 16*1024 {
		return PaymentContext{}, ErrUnavailable
	}
	var response struct {
		Data PaymentContext `json:"data"`
	}
	if json.Unmarshal(body, &response) != nil {
		return PaymentContext{}, ErrUnavailable
	}
	return response.Data, nil
}
