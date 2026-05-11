package models

import (
	"github.com/google/uuid"
	"time"
)

type Job struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	JobRef         string     `json:"job_ref" db:"job_ref"`
	EngagementType string     `json:"engagement_type" db:"engagement_type"`
	DurationMonths int        `json:"duration_months" db:"duration_months"`
	SalaryAmount   float64    `json:"salary_amount" db:"salary_amount"`
	Description    string     `json:"description" db:"description"`
	County         string     `json:"county" db:"county"`
	Residence      string     `json:"residence" db:"residence"`
	Requirements   string     `json:"requirements" db:"requirements"`
	Status         string     `json:"status" db:"status"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}
