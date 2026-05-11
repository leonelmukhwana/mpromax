package models

import (
	"github.com/google/uuid"
	"time"
)

// MonthlyRating represents the structured record in the database.
type MonthlyRating struct {
	ID           uuid.UUID `json:"id" db:"id"`
	AssignmentID uuid.UUID `json:"assignment_id" db:"assignment_id" validate:"required"`
	EmployerID   uuid.UUID `json:"employer_id" db:"employer_id" validate:"required"`
	NannyID      uuid.UUID `json:"nanny_id" db:"nanny_id" validate:"required"`

	// Rating Logic
	RatingValue int    `json:"rating_value" db:"rating_value" validate:"required,min=1,max=5"`
	ReviewText  string `json:"review_text" db:"review_text" validate:"required,max=500"`

	// Time-based Idempotency Keys
	RatingMonth int `json:"rating_month" db:"rating_month" validate:"required,min=1,max=12"`
	RatingYear  int `json:"rating_year" db:"rating_year" validate:"required"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
