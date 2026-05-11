package dto

import "time"

// VerificationResponse is what we send back to the frontend
// after a successful upload or when checking status.
type VerificationResponse struct {
	NannyID                string    `json:"nanny_id"`
	IDCardURL              string    `json:"id_card_url"`
	SelfieURL              string    `json:"selfie_url"`
	IsVerificationComplete bool      `json:"is_verification_complete"`
	VerifiedAt             time.Time `json:"verified_at"`
}

// VerificationStatusDTO is a lean version for quick checks
// (e.g., before showing the "Book Interview" button)
type VerificationStatusDTO struct {
	IsProfileComplete      bool `json:"is_profile_complete"`
	IsVerificationComplete bool `json:"is_verification_complete"`
	CanBookInterview       bool `json:"can_book_interview"`
}

// AdminVerifyUpdateDTO is used when the admin manually
// toggles the status after review.
type AdminVerifyUpdateDTO struct {
	IsComplete bool   `json:"is_complete" binding:"required"`
	Remarks    string `json:"remarks"` // Optional: Why they approved/rejected
}
