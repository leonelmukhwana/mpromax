package models

import (
	"github.com/google/uuid"
	"time"
)

// NannyAssignment represents the core domain entity.
// It applies the Snapshot Principle by storing salary and location
// to protect the contract integrity.
type NannyAssignment struct {
	ID         uuid.UUID `json:"id" db:"id"`
	JobID      uuid.UUID `json:"job_id" db:"job_id"`
	NannyID    uuid.UUID `json:"nanny_id" db:"nanny_id"`       // References users.id
	EmployerID uuid.UUID `json:"employer_id" db:"employer_id"` // References users.id

	// Contract Snapshots
	JobRef         string  `json:"job_ref" db:"job_ref"`
	County         string  `json:"county" db:"county"`
	Residence      string  `json:"residence" db:"residence"`
	SalaryAmount   float64 `json:"salary_amount" db:"salary_amount"`
	DurationMonths int     `json:"duration_months" db:"duration_months"`

	// State Management
	Status         string    `json:"status" db:"status"` // "active", "completed", "terminated"
	AssignmentDate time.Time `json:"assignment_date" db:"assignment_date"`
	// Add these two fields here:
	NannyName    string `json:"nanny_name"`
	EmployerName string `json:"employer_name"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsPaymentDue encapsulates the 30-day payroll business logic.
func (a *NannyAssignment) IsPaymentDue() bool {
	return time.Since(a.AssignmentDate).Hours() >= (24 * 30)
}
