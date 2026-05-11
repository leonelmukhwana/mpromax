package models

import (
	"github.com/google/uuid"
	"time"
)

type NannyVerification struct {
	ID                     uuid.UUID `json:"id" db:"id"`
	NannyID                uuid.UUID `json:"nanny_id" db:"nanny_id"`
	IDCardURL              string    `json:"id_card_url" db:"id_card_url"`
	SelfieURL              string    `json:"selfie_url" db:"selfie_url"`
	IDPublicID             string    `json:"id_public_id" db:"id_cloudinary_public_id"`
	SelfiePublicID         string    `json:"selfie_public_id" db:"selfie_cloudinary_public_id"`
	IsVerificationComplete bool      `json:"is_verification_complete" db:"is_verification_complete"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}
