package model

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

const ConfirmedEventType = "booking.confirmed.v1"
const InvoiceConsumer = "payment.booking-events.v1"

var ErrInvalidConfirmation = errors.New("invalid booking confirmation event")

type ConfirmedPayload struct {
	BookingID   string    `json:"booking_id"`
	PaymentID   string    `json:"payment_id"`
	CustomerID  string    `json:"customer_id"`
	ConfirmedAt time.Time `json:"confirmed_at"`
	TotalIDR    int64     `json:"total_idr"`
	Currency    string    `json:"currency"`
}

type ConfirmedEvent struct {
	EventID       string           `json:"event_id"`
	EventType     string           `json:"event_type"`
	SchemaVersion int              `json:"schema_version"`
	OccurredAt    time.Time        `json:"occurred_at"`
	AggregateID   string           `json:"aggregate_id"`
	CorrelationID string           `json:"correlation_id"`
	CausationID   string           `json:"causation_id"`
	Payload       ConfirmedPayload `json:"payload"`
	Hash          string           `json:"-"`
}

func DecodeConfirmedEvent(body []byte) (ConfirmedEvent, error) {
	var event ConfirmedEvent
	if len(body) > 16*1024 {
		return event, ErrInvalidConfirmation
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&event) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		return event, ErrInvalidConfirmation
	}
	if event.EventType != ConfirmedEventType || event.SchemaVersion != 1 || event.OccurredAt.IsZero() ||
		!event.OccurredAt.Equal(event.Payload.ConfirmedAt) || event.AggregateID != event.Payload.BookingID ||
		event.Payload.TotalIDR <= 0 || event.Payload.Currency != "IDR" || event.CorrelationID == "" || len(event.CorrelationID) > 64 ||
		strings.Trim(event.CorrelationID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_") != "" {
		return event, ErrInvalidConfirmation
	}
	for _, id := range []string{event.EventID, event.AggregateID, event.CausationID, event.Payload.PaymentID, event.Payload.CustomerID} {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed == uuid.Nil || len(id) != 36 {
			return event, ErrInvalidConfirmation
		}
	}
	digest := sha256.Sum256(body)
	event.Hash = hex.EncodeToString(digest[:])
	return event, nil
}
