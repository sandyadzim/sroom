package booking

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var ErrNotConfirmed = errors.New("booking has no verified confirmation")

type InvoiceContext struct {
	BookingID           string    `json:"booking_id"`
	CustomerID          string    `json:"customer_id"`
	VariantID           string    `json:"variant_id"`
	VehicleLabel        string    `json:"vehicle_label"`
	Status              string    `json:"status"`
	PaymentID           string    `json:"payment_id"`
	ProviderOrderID     string    `json:"provider_order_id"`
	ConfirmationEventID string    `json:"confirmation_event_id"`
	ConfirmedAt         time.Time `json:"confirmed_at"`
	PickupAt            time.Time `json:"pickup_at"`
	ReturnAt            time.Time `json:"return_at"`
	RentalDays          int       `json:"rental_days"`
	DailyRateIDR        int64     `json:"daily_rate_idr"`
	TotalIDR            int64     `json:"total_idr"`
	Currency            string    `json:"currency"`
}

func (c *Client) InvoiceContext(ctx context.Context, bookingID, requestID string) (InvoiceContext, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/internal/v1/bookings/"+bookingID+"/invoice-context", nil)
	if err != nil {
		return InvoiceContext{}, ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-Request-ID", requestID)
	res, err := c.http.Do(req)
	if err != nil {
		return InvoiceContext{}, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return InvoiceContext{}, ErrNotFound
	}
	if res.StatusCode == 409 {
		return InvoiceContext{}, ErrNotConfirmed
	}
	if res.StatusCode != 200 {
		return InvoiceContext{}, ErrUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 16*1024+1))
	if err != nil || len(body) > 16*1024 {
		return InvoiceContext{}, ErrUnavailable
	}
	var result struct {
		Data InvoiceContext `json:"data"`
	}
	if json.Unmarshal(body, &result) != nil {
		return InvoiceContext{}, ErrUnavailable
	}
	data := result.Data
	for _, id := range []string{data.BookingID, data.CustomerID, data.VariantID, data.PaymentID, data.ConfirmationEventID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return InvoiceContext{}, ErrUnavailable
		}
	}
	if data.BookingID != bookingID || data.Status != "confirmed" || data.ProviderOrderID == "" || data.VehicleLabel == "" || data.ConfirmedAt.IsZero() || data.PickupAt.IsZero() || !data.ReturnAt.After(data.PickupAt) || data.RentalDays < 1 || data.DailyRateIDR <= 0 || data.TotalIDR <= 0 || data.TotalIDR/int64(data.RentalDays) != data.DailyRateIDR || data.TotalIDR%int64(data.RentalDays) != 0 || data.Currency != "IDR" {
		return InvoiceContext{}, ErrUnavailable
	}
	return data, nil
}
