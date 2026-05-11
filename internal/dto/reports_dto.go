package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateIncidentRequest is what the Nanny or Employer sends to the API
type CreateIncidentRequest struct {
	AssignmentID uuid.UUID `json:"assignment_id" binding:"required"`
	ReportedID   uuid.UUID `json:"reported_id" binding:"required"`
	Subject      string    `json:"subject" binding:"required,max=255"`
	Description  string    `json:"description" binding:"required"`
}

// AdminIncidentResponse is the detailed view only for the Admin dashboard
type AdminIncidentResponse struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	Subject     string    `json:"subject"`
	Description string    `json:"description"`

	// Reporter Information
	ReporterName  string `json:"reporter_name"`
	ReporterEmail string `json:"reporter_email"`
	ReporterRole  string `json:"reporter_role"` // "nanny" or "employer"

	// Reported Party Information
	ReportedName  string `json:"reported_name"`
	ReportedEmail string `json:"reported_email"`

	// Context
	// Context & Feedback
	AssignmentID uuid.UUID `json:"assignment_id"`
	AdminNotes   string    `json:"admin_notes"` // <--- ADD THIS LINE
	CreatedAt    time.Time `json:"created_at"`
}

// IncidentListResponse wraps the list for the Admin API
type IncidentListResponse struct {
	Data  []AdminIncidentResponse `json:"data"`
	Total int                     `json:"total"`
}

// internal/dto/incident_dto.go

type IncidentFilterParams struct {
	Status       string `form:"status"`        // e.g., "pending"
	ReporterRole string `form:"reporter_role"` // e.g., "nanny"
	Search       string `form:"search"`        // Search by name/email
	Sort         string `form:"sort"`          // "asc" or "desc"
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

type UpdateIncidentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending investigating resolved dismissed"`
	Notes  string `json:"notes"`
}

// reporter vieew
type UserIncidentResponse struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"` // e.g., "pending", "resolved"
	Subject     string    `json:"subject"`
	Description string    `json:"description"`

	// Who did they report?
	ReportedName string `json:"reported_name"`

	// Feedback from the Admin
	AdminNotes string `json:"admin_notes"`

	AssignmentID uuid.UUID `json:"assignment_id"`
	CreatedAt    time.Time `json:"created_at"`
}
