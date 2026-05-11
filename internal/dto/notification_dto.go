package dto

import (
	"github.com/google/uuid"
	"time"
)

// SendNotificationRequest is the entry point for other services (Auth, Booking, etc.)
// to trigger a background notification.
type SendNotificationRequest struct {
	UserID    uuid.UUID              `json:"user_id" binding:"required"`
	EventType string                 `json:"event_type" binding:"required"` // e.g., "OTP_REQUEST", "JOB_ASSIGNED"
	Channels  []string               `json:"channels" binding:"required"`   // ["email", "push", "web"]
	Payload   map[string]interface{} `json:"payload" binding:"required"`    // Flexible data (OTP codes, Names, etc.)
	ExpiresIn *time.Duration         `json:"expires_in,omitempty"`          // Use for OTPs (e.g., 5*time.Minute)
}

// NotificationResponse is used when fetching notification history/status for a user
type NotificationResponse struct {
	ID            uuid.UUID `json:"id"`
	Channel       string    `json:"channel"`
	Status        string    `json:"status"`
	RetryCount    int       `json:"retry_count"`
	FormattedDate string    `json:"formatted_date"`
	LastError     string    `json:"last_error,omitempty"`
}

// NotificationListResponse handles paginated results
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Total         int64                  `json:"total"`
}
