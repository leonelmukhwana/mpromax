package dto

import (
	"github.com/google/uuid"
)

// --- SHARED COMPONENTS ---

// PaginationMeta provides metadata for paginated lists.
type PaginationMeta struct {
	TotalRecords int64 `json:"total_records"`
	TotalPages   int   `json:"total_pages"`
	CurrentPage  int   `json:"current_page"`
	PageSize     int   `json:"page_size"`
}

// RatingFilterParams handles selecting specific months/years and pagination.
type RatingFilterParams struct {
	Month int `form:"month" json:"month"`
	Year  int `form:"year" json:"year"`
	// Remove binding:"required" and use form tags correctly
	Page  int `form:"page" json:"page"`
	Limit int `form:"limit" json:"limit"`
}

// --- INPUT DTO ---

// CreateMonthlyRatingRequest is used by the Employer to submit a review.
type CreateMonthlyRatingRequest struct {
	AssignmentID uuid.UUID `json:"assignment_id" binding:"required"`
	NannyID      uuid.UUID `json:"nanny_id" binding:"required"`
	EmployerID   uuid.UUID `json:"employer_id"` // Set by middleware
	RatingValue  int       `json:"rating_value" binding:"required,min=1,max=5"`
	ReviewText   string    `json:"review_text" binding:"required,max=500"`
	Month        int       `json:"month" binding:"required,min=1,max=12"`
	Year         int       `json:"year" binding:"required"`
}

// --- NANNY VIEW DTOS ---

type NannyRatingListResponse struct {
	Data []NannyRatingItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

type NannyRatingItem struct {
	RatingValue           int     `json:"rating"`
	ReviewText            string  `json:"reason"`
	MonthName             string  `json:"month"` // e.g., "December"
	Year                  int     `json:"year"`
	MonthlyPerformancePct float64 `json:"performance_percentage"`
}

// --- ADMIN VIEW DTOS ---

// AdminRatingListResponse is the master list for administrators.
type AdminRatingListResponse struct {
	Data []AdminRatingItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

type AdminRatingItem struct {
	// Identity Info
	NannyFullName    string `json:"nanny_name"`
	NannyPhoneNumber string `json:"nanny_phone"`
	EmployerFullName string `json:"employer_name"`
	NannyEmail       string `json:"nanny_email"` // Add this

	// Historical Aggregate
	TotalAccumulatedScore int     `json:"total_accumulated_score"`
	LifetimePercentage    float64 `json:"lifetime_percentage"`

	// Snapshot Details (Filtered or Latest)
	CurrentMonthRating int    `json:"current_rating"`
	CurrentMonthReason string `json:"current_reason"`
	CurrentMonthName   string `json:"month_name"` // "January", etc.
	CurrentYear        int    `json:"year"`
}
