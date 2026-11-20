package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var ErrNotExpirable = errors.New("booking is not eligible for expiry assessment")

type ExpiryAssessment struct {
	BookingID           string     `json:"booking_id"`
	HoldExpiresAt       time.Time  `json:"hold_expires_at"`
	Result              string     `json:"result"`
	ClosedAt            *time.Time `json:"closed_at"`
	PaymentID           *string    `json:"payment_id"`
	ReleaseAuthorizedAt *time.Time `json:"release_authorized_at,omitempty"`
}

func (c *Client) AssessExpiry(ctx context.Context, bookingID, requestID string) (ExpiryAssessment, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/bookings/"+bookingID+"/expiry-assessment", nil)
	if err != nil {
		return ExpiryAssessment{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Request-ID", requestID)
	response, err := c.http.Do(req)
	if err != nil {
		return ExpiryAssessment{}, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusConflict {
		return ExpiryAssessment{}, ErrNotExpirable
	}
	if response.StatusCode != http.StatusOK {
		return ExpiryAssessment{}, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024+1))
	if err != nil || len(body) > 16*1024 {
		return ExpiryAssessment{}, ErrUnavailable
	}
	var result struct {
		Data ExpiryAssessment `json:"data"`
	}
	if json.Unmarshal(body, &result) != nil {
		return ExpiryAssessment{}, ErrUnavailable
	}
	data := result.Data
	if data.BookingID != bookingID || data.HoldExpiresAt.IsZero() {
		return ExpiryAssessment{}, ErrUnavailable
	}
	switch data.Result {
	case "no_attempt_closed":
		if data.ReleaseAuthorizedAt != nil || data.PaymentID != nil || data.ClosedAt == nil || data.ClosedAt.IsZero() || data.ClosedAt.Before(data.HoldExpiresAt) {
			return ExpiryAssessment{}, ErrUnavailable
		}
	case "provider_expired":
		if data.ClosedAt != nil || data.PaymentID == nil || data.ReleaseAuthorizedAt == nil ||
			data.ReleaseAuthorizedAt.IsZero() || data.ReleaseAuthorizedAt.Before(data.HoldExpiresAt) ||
			len(*data.PaymentID) != 36 || uuid.Validate(*data.PaymentID) != nil || *data.PaymentID == uuid.Nil.String() {
			return ExpiryAssessment{}, ErrUnavailable
		}
	case "attempt_exists":
		if data.ClosedAt != nil || data.PaymentID == nil || data.ReleaseAuthorizedAt != nil {
			return ExpiryAssessment{}, ErrUnavailable
		}
		id, err := uuid.Parse(*data.PaymentID)
		if err != nil || id == uuid.Nil || len(*data.PaymentID) != 36 {
			return ExpiryAssessment{}, ErrUnavailable
		}
	default:
		return ExpiryAssessment{}, ErrUnavailable
	}
	return data, nil
}
