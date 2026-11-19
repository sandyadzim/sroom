package model

import (
	"encoding/json"
	"time"
)

type OutboxEvent struct {
	ID            string
	AggregateID   string
	EventType     string
	Envelope      json.RawMessage
	OccurredAt    time.Time
	CorrelationID string `gorm:"-"`
}
