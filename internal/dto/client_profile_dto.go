package dto

import (
	"errors"
	"strings"
	"time"
)

// CreateClientProfileDTO: Logic for "Either/Or" ID/Passport
type CreateClientProfileDTO struct {
	FullName       string  `json:"full_name" binding:"required"`
	IDNumber       *string `json:"id_number"`
	PassportNumber *string `json:"passport_number"`
	PhoneNumber    string  `json:"phone_number" binding:"required"`
	Gender         string  `json:"gender" binding:"required,oneof=male female other"`
	Nationality    string  `json:"nationality" binding:"required"`
	County         string  `json:"county" binding:"required"`
	Residence      string  `json:"residence" binding:"required"`
}

func (d *CreateClientProfileDTO) Validate() error {
	// 1. Extract values or empty strings to check content, not just pointers
	idVal := ""
	if d.IDNumber != nil {
		idVal = strings.TrimSpace(*d.IDNumber)
	}

	passVal := ""
	if d.PassportNumber != nil {
		passVal = strings.TrimSpace(*d.PassportNumber)
	}

	// 2. Business Rule: Must have exactly one
	if idVal == "" && passVal == "" {
		return errors.New("provide either id_number or passport_number")
	}
	if idVal != "" && passVal != "" {
		return errors.New("only one of id_number or passport_number allowed")
	}

	return nil
}

// UpdateClientProfileDTO
type UpdateClientProfileDTO struct {
	PhoneNumber *string `json:"phone_number"`
	Nationality *string `json:"nationality"`
	County      *string `json:"county"`
	Residence   *string `json:"residence"`
}

func (d *UpdateClientProfileDTO) Validate() error {
	if d.PhoneNumber == nil && d.Nationality == nil && d.County == nil && d.Residence == nil {
		return errors.New("at least one field must be provided for update")
	}

	// Helper to ensure provided pointers aren't just empty spaces
	sanitize := func(p *string) error {
		if p != nil && strings.TrimSpace(*p) == "" {
			return errors.New("provided fields cannot be empty")
		}
		return nil
	}

	fields := []*string{d.PhoneNumber, d.Nationality, d.County, d.Residence}
	for _, f := range fields {
		if err := sanitize(f); err != nil {
			return err
		}
	}
	return nil
}

// ClientProfileResponse: Uses omitempty to hide whichever field is null
type ClientProfileResponse struct {
	ID             string  `json:"id"`
	FullName       string  `json:"full_name"`
	PhoneNumber    string  `json:"phone_number"`
	IDNumber       *string `json:"id_number,omitempty"`
	PassportNumber *string `json:"passport_number,omitempty"`
	Gender         string  `json:"gender"`
	Nationality    string  `json:"nationality"`
	County         string  `json:"county"`
	Residence      string  `json:"residence"`
	FCMToken       string  `json:"fcm_token"`

	CreatedAt time.Time `json:"created_at"`
}

// AdminClientResponse: Admins usually see all sensitive fields
type AdminClientResponse struct {
	ID             string    `json:"id"`
	FullName       string    `json:"full_name"`
	Gender         string    `json:"gender"`
	PhoneNumber    string    `json:"phone_number"`
	IDNumber       *string   `json:"id_number,omitempty"`
	PassportNumber *string   `json:"passport_number,omitempty"`
	Nationality    string    `json:"nationality"`
	County         string    `json:"county"`
	Residence      string    `json:"residence"`
	FCMToken       string    `json:"fcm_token"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminListClientsQuery struct {
	Limit  int     `form:"limit"`
	Cursor *string `form:"cursor"`
	Search string  `form:"search"`
	County *string `form:"county"`
	Gender *string `form:"gender"`
}

func (q *AdminListClientsQuery) Normalize() {
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}
	q.Search = strings.TrimSpace(q.Search)
}

type AdminListClientsResponse struct {
	Data       []AdminClientResponse `json:"data"`
	NextCursor *time.Time            `json:"next_cursor"`
}

type DeleteProfileResponse struct {
	Message   string    `json:"message"`
	DeletedAt time.Time `json:"deleted_at"`
}

type RecoverProfileResponse struct {
	Message string `json:"message"`
}
