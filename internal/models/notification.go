package models

import (
	"github.com/google/uuid"
	"time"
)

type NotificationRequest struct {
	UserID    uuid.UUID           `json:"user_id"`
	EventType string              `json:"event_type"`
	Channels  []string            `json:"channels"`
	Payload   NotificationPayload `json:"payload"`
}

type NotificationPayload struct {
	Title    string         `json:"title"`
	Body     string         `json:"body"`
	Metadata map[string]any `json:"metadata"`
}

type Notification struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	Title     string                 `json:"title" db:"title"`
	Message   string                 `json:"message" db:"message"`
	Channel   string                 `json:"channel" db:"channel"` // email, push, web
	Type      string                 `json:"type" db:"type"`       // otp, payment, etc.
	Status    string                 `json:"status" db:"status"`   // pending, sent, failed
	Retries   int                    `json:"retries" db:"retries"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
}
