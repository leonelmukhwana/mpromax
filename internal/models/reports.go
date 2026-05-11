package models

import (
	"time"

	"github.com/google/uuid"
)

// IncidentReport represents the raw database record
type IncidentReport struct {
	ID           uuid.UUID `json:"id" db:"id"`
	AssignmentID uuid.UUID `json:"assignment_id" db:"assignment_id"`

	// Who is reporting
	ReporterID   uuid.UUID `json:"reporter_id" db:"reporter_id"`
	ReporterRole string    `json:"reporter_role" db:"reporter_role"` // "nanny" or "employer"

	// Who is being reported
	ReportedID uuid.UUID `json:"reported_id" db:"reported_id"`

	// Details
	Subject     string `json:"subject" db:"subject"`
	Description string `json:"description" db:"description"`

	// Status Management
	Status string `json:"status" db:"status"` // "pending", "investigating", "resolved"

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
