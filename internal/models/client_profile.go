package models

import (
	"time"

	"github.com/google/uuid"
)

type ClientProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key"`
	UserID uuid.UUID `gorm:"type:uuid;not null"`

	FullName string

	IDNumberEncrypted       *string
	IDNumberHash            *string
	PassportNumberEncrypted *string
	PassportNumberHash      *string

	PhoneEncrypted string
	PhoneHash      string

	Gender      string
	Nationality string
	County      string
	Residence   string

	IsDeleted bool
	DeletedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time // Fixed: Added "Time" and removed the trailing dot
	FCMToken  string    `gorm:"column:fcm_token" json:"fcm_token"`
}
