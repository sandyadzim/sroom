package payment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidEvent = errors.New("invalid payment event")

const PaidEventType = "payment.paid.v1"

type PaidEvent struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version"`
	OccurredAt    time.Time       `json:"occurred_at"`
	AggregateID   string          `json:"aggregate_id"`
	CorrelationID string          `json:"correlation_id"`
	CausationID   json.RawMessage `json:"causation_id"`
	Payload       PaidPayload     `json:"payload"`
	Hash          string          `json:"-"`
}

type PaidPayload struct {
	PaymentID       string          `json:"payment_id"`
	BookingID       string          `json:"booking_id"`
	ProviderOrderID string          `json:"provider_order_id"`
	AmountIDR       int64           `json:"amount_idr"`
	Currency        string          `json:"currency"`
	VerifiedAt      time.Time       `json:"verified_at"`
	ProviderPaidAt  json.RawMessage `json:"provider_paid_at"`
}

func DecodePaidEvent(body []byte) (PaidEvent, error) {
	var event PaidEvent
	if len(body) > 16*1024 {
		return event, ErrInvalidEvent
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&event) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		return event, ErrInvalidEvent
	}
	if event.EventType != PaidEventType || event.SchemaVersion != 1 || event.OccurredAt.IsZero() || event.Payload.VerifiedAt.IsZero() ||
		event.AggregateID != event.Payload.PaymentID || event.Payload.AmountIDR <= 0 || event.Payload.Currency != "IDR" ||
		strings.TrimSpace(event.Payload.ProviderOrderID) == "" || len(event.Payload.ProviderOrderID) > 50 ||
		event.CorrelationID == "" || len(event.CorrelationID) > 64 || strings.Trim(event.CorrelationID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_") != "" ||
		len(event.CausationID) == 0 || len(event.Payload.ProviderPaidAt) == 0 {
		return event, ErrInvalidEvent
	}
	for _, id := range []string{event.EventID, event.AggregateID, event.Payload.BookingID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return event, ErrInvalidEvent
		}
	}
	if string(event.CausationID) != "null" {
		var id string
		if json.Unmarshal(event.CausationID, &id) != nil {
			return event, ErrInvalidEvent
		}
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return event, ErrInvalidEvent
		}
	}
	if string(event.Payload.ProviderPaidAt) != "null" {
		var at time.Time
		if json.Unmarshal(event.Payload.ProviderPaidAt, &at) != nil || at.IsZero() {
			return event, ErrInvalidEvent
		}
	}
	digest := sha256.Sum256(body)
	event.Hash = hex.EncodeToString(digest[:])
	return event, nil
}
