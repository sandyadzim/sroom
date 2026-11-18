package catalog

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

var (
	ErrVariantNotFound  = errors.New("catalog variant not found")
	ErrInvalidVariantID = errors.New("variant ID must be a UUID")
	ErrUnavailable      = errors.New("catalog is unavailable")
)

type BookingSnapshot struct {
	VariantID        string   `json:"variant_id"`
	VehicleLabel     string   `json:"vehicle_label"`
	DailyRateIDR     int64    `json:"daily_rate_idr"`
	Status           string   `json:"status"`
	EligibleUnitIDs  []string `json:"eligible_unit_ids"`
	DemoHoldsEnabled bool     `json:"demo_holds_enabled"`
}

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewClient(baseURL, token string) (*Client, error) {
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Hostname() == "" ||
		base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" ||
		(base.Path != "" && base.Path != "/") {
		return nil, errors.New("CATALOG_BASE_URL must be an HTTP origin")
	}
	token = strings.TrimSpace(token)
	if len(strings.Fields(token)) != 1 {
		return nil, errors.New("BOOKING_CATALOG_TOKEN must be a single token")
	}
	return &Client{
		baseURL: strings.TrimRight(base.String(), "/"),
		token:   token,
		http: &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) BookingSnapshot(ctx context.Context, variantID, requestID string) (BookingSnapshot, error) {
	id, err := uuid.Parse(variantID)
	if err != nil || len(variantID) != 36 {
		return BookingSnapshot{}, ErrInvalidVariantID
	}
	endpoint := c.baseURL + "/internal/v1/variants/" + id.String() + "/booking-snapshot"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return BookingSnapshot{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}

	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return BookingSnapshot{}, ctx.Err()
		}
		return BookingSnapshot{}, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return BookingSnapshot{}, ErrUnavailable
	}

	const maxResponseBytes = 64 * 1024
	body, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return BookingSnapshot{}, ctx.Err()
		}
		return BookingSnapshot{}, ErrUnavailable
	}
	var result struct {
		Data  BookingSnapshot `json:"data"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if len(body) > maxResponseBytes || json.Unmarshal(body, &result) != nil {
		return BookingSnapshot{}, ErrUnavailable
	}
	if res.StatusCode == http.StatusNotFound && result.Error.Code == "CAR_NOT_FOUND" {
		return BookingSnapshot{}, ErrVariantNotFound
	}
	if res.StatusCode != http.StatusOK || !result.Data.validFor(id.String()) {
		return BookingSnapshot{}, ErrUnavailable
	}
	return result.Data, nil
}

func (s BookingSnapshot) validFor(id string) bool {
	if s.VariantID != id || strings.TrimSpace(s.VehicleLabel) == "" || s.DailyRateIDR <= 0 || s.EligibleUnitIDs == nil {
		return false
	}
	switch s.Status {
	case "inactive":
		return len(s.EligibleUnitIDs) == 0
	case "active":
		for _, unitID := range s.EligibleUnitIDs {
			if len(unitID) != 36 || uuid.Validate(unitID) != nil {
				return false
			}
		}
		return true
	default:
		return false
	}
}
