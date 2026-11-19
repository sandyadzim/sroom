package model

import "time"

const MaxEmailAttempts = 5

type EmailDelivery struct {
	ID            string
	InvoiceID     string
	BookingID     string
	Kind          string
	Recipient     string
	Status        string
	AttemptCount  int
	NextAttemptAt *time.Time
	LockedUntil   *time.Time
	MessageID     string
	SentAt        *time.Time
	LastErrorCode *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
