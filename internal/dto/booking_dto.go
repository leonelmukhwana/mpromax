package dto

import (
	"github.com/google/uuid"
	"time"
)

// CreateBookingRequest is what the Nanny sends from the frontend
type CreateBookingRequest struct {
	// We don't include NannyID here because we extract it from the Auth Token for security
	BookingSlot    time.Time `json:"booking_slot" binding:"required" example:"2026-04-25T08:30:00Z"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required" example:"unique-uuid-per-click"`
}

// BookingResponse is a clean version of the booking for the UI
type BookingResponse struct {
	ID          uuid.UUID         `json:"id"`
	BookingSlot time.Time         `json:"booking_slot"`
	Nanny       NannyAdminDetails `json:"nanny"`

	Status    string    `json:"status"` // e.g., "Confirmed"
	CreatedAt time.Time `json:"created_at"`
}

type NannyAdminDetails struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}
type AdminBookingResponse struct {
	ID            uuid.UUID         `json:"id"`
	BookingSlot   time.Time         `json:"booking_slot"`   // Keep raw for logic
	FormattedSlot string            `json:"formatted_slot"` // "April 26, 2026 - 12:00 PM"
	Nanny         NannyAdminDetails `json:"nanny"`
	Status        string            `json:"status"` // Added this
	CreatedAt     time.Time         `json:"created_at"`
}

// AdminBookingFilter is used for the "best search algorithm" pagination/filtering
type AdminBookingFilter struct {
	NannyName string    `form:"nanny_name"` // Search by nanny name
	FromDate  time.Time `form:"from_date" time_format:"2006-01-02"`
	ToDate    time.Time `form:"to_date" time_format:"2006-01-02"`
	Page      int       `form:"page,default=1"`
	Limit     int       `form:"limit,default=10"`
}

// PaginatedBookingResponse for Admin dashboard
type PaginatedBookingResponse struct {
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
	Items []BookingResponse `json:"items"`
}
