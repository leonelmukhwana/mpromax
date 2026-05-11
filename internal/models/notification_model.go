package models

import (
	"github.com/google/uuid"
	"time"
)

type NotificationOutbox struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	EventType string                 `json:"event_type"` // e.g., "BOOKING_CREATED"
	Payload   map[string]interface{} `json:"payload"`    // Title, Message, etc.

	// Separate status for each channel in one row
	EmailStatus string `json:"email_status"` // pending, sent, failed, skipped
	PushStatus  string `json:"push_status"`  // pending, sent, failed, skipped
	WebStatus   string `json:"web_status"`   // pending, sent, failed, skipped

	RetryCount  int        `json:"retry_count"`
	NextRetryAt time.Time  `json:"next_retry_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastError   string     `json:"last_error"`
	CreatedAt   time.Time  `json:"created_at"`
}
