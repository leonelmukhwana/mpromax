package dto

import (
	"time"

	"github.com/google/uuid"
)

// Create Assignment Request
type CreateAssignmentRequest struct {
	JobID   uuid.UUID `json:"job_id" validate:"required"`
	NannyID uuid.UUID `json:"nanny_id" validate:"required"`
}

// Admin Response (FULL)
type AssignmentResponse struct {
	ID             uuid.UUID `json:"id"`
	JobID          uuid.UUID `json:"job_id"`
	NannyID        uuid.UUID `json:"nanny_id"`
	ClientID       uuid.UUID `json:"client_id"`
	JobRef         string    `json:"job_ref"`
	County         string    `json:"county"`
	Residence      string    `json:"residence"`
	SalaryAmount   float64   `json:"salary_amount"`
	DurationMonths int       `json:"duration_months"`
	Status         string    `json:"status"`
	AssignmentDate time.Time `json:"assignment_date"`
}

// Nanny View (NO SALARY)
type NannyAssignmentResponse struct {
	ID             uuid.UUID `json:"id"`
	JobRef         string    `json:"job_ref"`
	County         string    `json:"county"`
	Residence      string    `json:"residence"`
	DurationMonths int       `json:"duration_months"`
	Status         string    `json:"status"`
	AssignmentDate time.Time `json:"assignment_date"`
	EmployerName   string    `json:"employer_name"`
}

// Filtering + Pagination
type AssignmentFilter struct {
	Search   string
	Status   string
	Page     int
	PageSize int
}
