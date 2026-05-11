package dto

type NotificationRequest struct {
	UserID  string `json:"user_id" validate:"required"`
	Title   string `json:"title" validate:"required"`
	Message string `json:"message" validate:"required"`
	Channel string `json:"channel"` // Optional: defaults to user preference
}
