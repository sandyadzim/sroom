package service

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"payment/internal/model"

	"github.com/google/uuid"
)

// Only the current payment.paid.v1 contract is supported. Do not publish arbitrary DB JSON.
func validateOutboxEvent(event *model.OutboxEvent) bool {
	var envelope struct {
		EventID       string          `json:"event_id"`
		EventType     string          `json:"event_type"`
		SchemaVersion int             `json:"schema_version"`
		OccurredAt    time.Time       `json:"occurred_at"`
		AggregateID   string          `json:"aggregate_id"`
		CorrelationID string          `json:"correlation_id"`
		CausationID   json.RawMessage `json:"causation_id"`
		Payload       struct {
			PaymentID       string          `json:"payment_id"`
			BookingID       string          `json:"booking_id"`
			ProviderOrderID string          `json:"provider_order_id"`
			AmountIDR       int64           `json:"amount_idr"`
			Currency        string          `json:"currency"`
			VerifiedAt      time.Time       `json:"verified_at"`
			ProviderPaidAt  json.RawMessage `json:"provider_paid_at"`
		} `json:"payload"`
	}
	decoder := json.NewDecoder(bytes.NewReader(event.Envelope))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		return false
	}
	if envelope.EventID != event.ID || envelope.AggregateID != event.AggregateID || envelope.EventType != event.EventType ||
		envelope.EventType != "payment.paid.v1" || envelope.SchemaVersion != 1 || envelope.OccurredAt.IsZero() ||
		!envelope.OccurredAt.Equal(event.OccurredAt) || envelope.Payload.PaymentID != event.AggregateID ||
		envelope.Payload.AmountIDR <= 0 || envelope.Payload.Currency != "IDR" || envelope.Payload.VerifiedAt.IsZero() ||
		strings.TrimSpace(envelope.Payload.ProviderOrderID) == "" || utf8.RuneCountInString(envelope.Payload.ProviderOrderID) > 50 ||
		envelope.CorrelationID == "" || len(envelope.CorrelationID) > 64 {
		return false
	}
	for _, id := range []string{envelope.EventID, envelope.AggregateID, envelope.Payload.BookingID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return false
		}
	}
	if len(envelope.CausationID) == 0 || len(envelope.Payload.ProviderPaidAt) == 0 {
		return false
	}
	if string(envelope.CausationID) != "null" {
		var id string
		if json.Unmarshal(envelope.CausationID, &id) != nil {
			return false
		}
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return false
		}
	}
	if string(envelope.Payload.ProviderPaidAt) != "null" {
		var paidAt time.Time
		if json.Unmarshal(envelope.Payload.ProviderPaidAt, &paidAt) != nil || paidAt.IsZero() {
			return false
		}
	}
	event.CorrelationID = envelope.CorrelationID
	return true
}
