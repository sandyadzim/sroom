package service

import (
	"encoding/json"
	"testing"
	"time"

	"payment/internal/model"

	"github.com/google/uuid"
)

func syntheticOutboxEvent(t *testing.T) model.OutboxEvent {
	t.Helper()
	event := model.OutboxEvent{ID: uuid.NewString(), AggregateID: uuid.NewString(), EventType: "payment.paid.v1", OccurredAt: time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC)}
	envelope := map[string]any{"event_id": event.ID, "aggregate_id": event.AggregateID, "event_type": event.EventType,
		"schema_version": 1, "occurred_at": event.OccurredAt, "correlation_id": "test-outbox", "causation_id": nil,
		"payload": map[string]any{"payment_id": event.AggregateID, "booking_id": uuid.NewString(), "provider_order_id": "synthetic-order", "amount_idr": 150000, "currency": "IDR", "verified_at": event.OccurredAt, "provider_paid_at": nil}}
	var err error
	event.Envelope, err = json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func TestOutboxEventContract(t *testing.T) {
	for _, name := range []string{"valid", "wrong id", "wrong aggregate", "wrong type", "wrong version", "wrong time", "missing cause", "invalid cause", "missing paid time", "invalid paid time", "invalid booking", "amount", "currency", "unknown field", "unknown payload field", "blank correlation", "trailing JSON"} {
		t.Run(name, func(t *testing.T) {
			event := syntheticOutboxEvent(t)
			var envelope map[string]any
			if err := json.Unmarshal(event.Envelope, &envelope); err != nil {
				t.Fatal(err)
			}
			payload := envelope["payload"].(map[string]any)
			switch name {
			case "wrong id":
				envelope["event_id"] = uuid.NewString()
			case "wrong aggregate":
				payload["payment_id"] = uuid.NewString()
			case "wrong type":
				envelope["event_type"] = "booking.demo_confirmed.v1"
			case "wrong version":
				envelope["schema_version"] = 2
			case "wrong time":
				envelope["occurred_at"] = event.OccurredAt.Add(time.Minute)
			case "missing cause":
				delete(envelope, "causation_id")
			case "invalid cause":
				envelope["causation_id"] = "invalid"
			case "missing paid time":
				delete(payload, "provider_paid_at")
			case "invalid paid time":
				payload["provider_paid_at"] = "invalid"
			case "invalid booking":
				payload["booking_id"] = "invalid"
			case "amount":
				payload["amount_idr"] = 0
			case "currency":
				payload["currency"] = "USD"
			case "unknown field":
				envelope["signature"] = "must-not-publish"
			case "unknown payload field":
				payload["email"] = "must-not-publish"
			case "blank correlation":
				envelope["correlation_id"] = ""
			}
			var err error
			event.Envelope, err = json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if name == "trailing JSON" {
				event.Envelope = append(event.Envelope, []byte("{}")...)
			}
			if valid := validateOutboxEvent(&event); valid != (name == "valid") {
				t.Fatalf("valid=%v", valid)
			}
			if name == "valid" && event.CorrelationID != "test-outbox" {
				t.Fatal("missing correlation")
			}
		})
	}
}
