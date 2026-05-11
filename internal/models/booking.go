package models

import (
	"time"

	"github.com/google/uuid"
)

// Booking represents the DB schema and the API response
type Booking struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	NannyID        uuid.UUID `gorm:"type:uuid;not null;unique" json:"nanny_id"`
	BookingSlot    time.Time `gorm:"type:timestamptz;not null;unique" json:"booking_slot"`
	IdempotencyKey string    `gorm:"type:varchar(255);not null;unique" json:"idempotency_key"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// Optional: If you want to load Nanny details in the Admin search
	Nanny *User `gorm:"foreignKey:NannyID" json:"nanny,omitempty"`
}

// BookingRequest defines what the frontend sends
type BookingRequest struct {
	// The user_id will actually come from the JWT/Context, not the body
	BookingSlot    time.Time `json:"booking_slot" binding:"required"`
	IdempotencyKey string    `json:"idempotency_key" binding:"required"`
}

// BookingResponse for the Admin paginated list
type BookingPaginationResponse struct {
	Data    []Booking `json:"data"`
	Total   int64     `json:"total"`
	Page    int       `json:"page"`
	Limit   int       `json:"limit"`
	HasMore bool      `json:"has_more"`
}

// AdminBookingFilter handles query parameters for the admin dashboard
type AdminBookingFilter struct {
	// NannyName allows searching by the nanny's full name (ILIKE search)
	NannyName string `form:"nanny_name"`

	// Date filters for narrowing down the interview schedule
	FromDate time.Time `form:"from_date" time_format:"2006-01-02"`
	ToDate   time.Time `form:"to_date" time_format:"2006-01-02"`

	// Pagination parameters
	// default values are handled in the controller or repo
	Page  int `form:"page,default=1"`
	Limit int `form:"limit,default=10"`
}
