package models

import (
	"time"

	"github.com/google/uuid"
)

// NannyProfile represents the database structure
type NannyProfile struct {
	UserID         uuid.UUID  `json:"user_id"`
	FullName       string     `json:"full_name"`
	IDNumber       string     `json:"id_number"`    // Plaintext (decrypted)
	PhoneNumber    string     `json:"phone_number"` // Plaintext (decrypted)
	DOB            time.Time  `json:"dob"`
	Age            int        `json:"age"` // Calculated field
	HomeCounty     string     `json:"home_county"`
	EducationLevel string     `json:"education_level"`
	SelfieURL      string     `json:"selfie_url"`
	IDCardURL      string     `json:"id_card_url"`
	IsVerified     bool       `json:"is_verified"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Version        int        `json:"version"`
	FCMToken       string     `gorm:"column:fcm_token" json:"fcm_token"`
}

// CalculateAge computes age from DOB
func (n *NannyProfile) CalculateAge() {
	now := time.Now()
	years := now.Year() - n.DOB.Year()
	if now.YearDay() < n.DOB.YearDay() {
		years--
	}
	n.Age = years
}

// NannySearchFilter is for Admin pagination and filtering
// This fixes your "undefined" error in the Repository
type NannySearchFilter struct {
	Name   string `form:"name"`
	County string `form:"county"`
	Page   int    `form:"page"`
	Limit  int    `form:"limit"`
}
