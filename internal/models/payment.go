package models

import (
	"github.com/google/uuid"
	"time"
)

type Payment struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	AssignmentID     uuid.UUID `gorm:"not null"`
	JobRef           string    `gorm:"not null"`
	MpesaReceipt     string    `gorm:"unique"`
	TotalAmount      float64   `gorm:"type:numeric(15,2)"`
	CommissionAmount float64   `gorm:"type:numeric(15,2)"`
	NetAmount        float64   `gorm:"type:numeric(15,2)"`
	Status           string    `gorm:"type:payment_status"`
	BillingMonth     int
	BillingYear      int
	IdempotencyKey   string `gorm:"unique"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type LedgerEntry struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	PaymentID uuid.UUID
	Account   string `gorm:"type:account_type"`
	Debit     float64
	Credit    float64
	CreatedAt time.Time
}

type AssignmentDetail struct {
	ID             uuid.UUID
	EmployerID     uuid.UUID
	NannyID        uuid.UUID
	SalaryAmount   float64
	JobRef         string    // Added: Needed for identification
	AssignmentDate time.Time // Added: The "Anchor" for the 30-day clock
}
