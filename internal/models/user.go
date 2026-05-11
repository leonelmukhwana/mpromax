package models

import (
	"github.com/google/uuid"
	"time"
)

type Role string

const (
	RoleNanny    Role = "nanny"
	RoleEmployer Role = "employer"
	RoleAdmin    Role = "admin"
)

type Status string

const (
	StatusActive  Status = "active"
	StatusBlocked Status = "blocked"
	StatusPending Status = "pending"
)

type User struct {
	ID             uuid.UUID  `json:"id"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	Role           Role       `json:"role"`
	Status         Status     `json:"status"`
	FailedAttempts int        `json:"failed_attempts"`
	LockedUntil    *time.Time `json:"locked_until"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}

type OTP struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Code      string    `json:"code"`
	Type      string    `json:"type"` // email_verification, admin_2fa, password_reset
	Attempts  int       `json:"attempts"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
