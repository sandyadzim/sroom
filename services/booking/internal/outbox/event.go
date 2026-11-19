package outbox

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"booking/internal/checkout"

	"github.com/google/uuid"
)

func validateOutboxEvent(event *checkout.OutboxEvent) bool {
	var envelope struct {
		EventID       string    `json:"event_id"`
		EventType     string    `json:"event_type"`
		SchemaVersion int       `json:"schema_version"`
		OccurredAt    time.Time `json:"occurred_at"`
		AggregateID   string    `json:"aggregate_id"`
		CorrelationID string    `json:"correlation_id"`
		CausationID   string    `json:"causation_id"`
		Payload       struct {
			BookingID   string    `json:"booking_id"`
			PaymentID   string    `json:"payment_id"`
			CustomerID  string    `json:"customer_id"`
			ConfirmedAt time.Time `json:"confirmed_at"`
			TotalIDR    int64     `json:"total_idr"`
			Currency    string    `json:"currency"`
		} `json:"payload"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Envelope))
	decoder.DisallowUnknownFields()
	if len(event.Envelope) > 16*1024 || decoder.Decode(&envelope) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		return false
	}
	if envelope.EventID != event.ID || envelope.AggregateID != event.AggregateID || envelope.EventType != event.EventType || envelope.EventType != "booking.confirmed.v1" || envelope.SchemaVersion != 1 || envelope.OccurredAt.IsZero() || !envelope.OccurredAt.Equal(event.OccurredAt) || !envelope.OccurredAt.Equal(envelope.Payload.ConfirmedAt) || envelope.Payload.BookingID != event.AggregateID || envelope.Payload.TotalIDR <= 0 || envelope.Payload.Currency != "IDR" || envelope.CorrelationID == "" || len(envelope.CorrelationID) > 64 || strings.Trim(envelope.CorrelationID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_") != "" {
		return false
	}
	for _, id := range []string{envelope.EventID, envelope.AggregateID, envelope.CausationID, envelope.Payload.PaymentID, envelope.Payload.CustomerID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return false
		}
	}
	event.CorrelationID = envelope.CorrelationID
	return true
}
