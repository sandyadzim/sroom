package dto

import "time"

type EmailDeliveryStatus struct {
	Kind          string     `json:"kind"`
	Status        string     `json:"status"`
	AttemptCount  int        `json:"attempt_count"`
	NextAttemptAt *time.Time `json:"next_attempt_at"`
	SentAt        *time.Time `json:"sent_at"`
	LastErrorCode *string    `json:"last_error_code"`
}
