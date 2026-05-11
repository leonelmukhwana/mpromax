package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- 1. NANNY REGISTRATION (POST /profiles) ---
type NannyProfileRequest struct {
	FullName       string `json:"full_name" validate:"required,min=3,max=100"`
	IDNumber       string `json:"id_number" validate:"required"`
	PhoneNumber    string `json:"phone_number" validate:"required"` // Format: 07... or 01...
	DOB            string `json:"dob" validate:"required"`          // Expected: "YYYY-MM-DD"
	HomeCounty     string `json:"home_county" validate:"required"`
	EducationLevel string `json:"education_level" validate:"required"`
}

// --- 2. ADMIN LIST VIEW (GET /admin/nannies) ---
// High-level overview for the Admin dashboard table
type NannyListResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	FullName    string    `json:"full_name"`
	IDNumber    string    `json:"id_number"`    // Decrypted for quick manual call
	PhoneNumber string    `json:"phone_number"` // Decrypted for quick manual call
	HomeCounty  string    `json:"home_county"`
	Age         int       `json:"age"`

	EducationLevel string `json:"education_level"`
	IsVerified     bool   `json:"is_verified"`
	Status         string `json:"status"` // "Active", "Pending", or "Soft-Deleted"
}

// Full details including sensitive IDs for manual verification
type NannyAdminViewResponse struct {
	UserID         uuid.UUID  `json:"user_id"`
	FullName       string     `json:"full_name"`
	IDNumber       string     `json:"id_number"`    // Decrypted for Admin
	PhoneNumber    string     `json:"phone_number"` // Decrypted for Admin
	DOB            time.Time  `json:"dob"`
	Age            int        `json:"age"`
	HomeCounty     string     `json:"home_county"`
	EducationLevel string     `json:"education_level"`
	IsVerified     bool       `json:"is_verified"`
	FCMToken       string     `json:"fcm_token"`
	CreatedAt      time.Time  `json:"registered_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	RecoveryWindow string     `json:"recovery_window,omitempty"` // e.g., "45 days remaining"
}

// --- 4. NANNY SELF VIEW (GET /profile/me) ---
type NannySelfViewResponse struct {
	FullName       string `json:"full_name"`
	PhoneNumber    string `json:"phone_number"`
	IDNumber       string `json:"id_number"`
	Age            int    `json:"age"`
	HomeCounty     string `json:"home_county"`
	EducationLevel string `json:"education_level"`
	IsVerified     bool   `json:"is_verified"`
}

// --- 5. PAGINATION & WRAPPERS ---
type PaginatedNannyResponse struct {
	Data       []NannyListResponse `json:"nannies"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	TotalPages int                 `json:"total_pages"`
}

type NannyUpdateProfileRequest struct {
	PhoneNumber    *string `json:"phone_number" validate:"omitempty"`
	HomeCounty     *string `json:"home_county" validate:"omitempty"`
	EducationLevel *string `json:"education_level" validate:"omitempty"`
	DOB            *string `json:"dob" validate:"omitempty"` // "YYYY-MM-DD"
}

// Even for a soft delete, we capture a reason for the Audit Log.
type NannyDeleteRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=255"`
}

// As requested, no notes required—just the action trigger.
type NannyRecoverRequest struct {
	// Effectively empty, but kept as a struct for future-proofing
	// or to pass a confirmation boolean if needed.
}
