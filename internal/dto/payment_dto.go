package dto

import (
	"github.com/google/uuid"
	"time"
)

// PaymentResponse represents the master structure for Admin
type PaymentResponse struct {
	ID               uuid.UUID `json:"id"`
	AssignmentID     uuid.UUID `json:"assignment_id"`
	JobRef           string    `json:"job_ref"`
	MpesaReceipt     string    `json:"mpesa_receipt"`
	EmployerName     string    `json:"employer_name"`
	NannyName        string    `json:"nanny_name"`
	TotalAmount      float64   `json:"total_amount"`      // 100%
	CommissionAmount float64   `json:"commission_amount"` // 10%
	NetAmount        float64   `json:"net_amount"`        // 90%
	Status           string    `json:"status"`
	BillingMonth     string    `json:"billing_month"` // Human readable (e.g., "May")
	BillingYear      int       `json:"billing_year"`
	CreatedAt        time.Time `json:"created_at"`
}

// EmployerPaymentView is what the employer sees in their table
type EmployerPaymentView struct {
	NannyName    string    `json:"nanny_name"`
	AmountPaid   float64   `json:"amount_paid"` // The salary_amount (100%)
	Month        string    `json:"month"`
	Status       string    `json:"status"`
	MpesaReceipt string    `json:"mpesa_receipt"`
	CreatedAt    time.Time `json:"date"`
}

// NannyPaymentView is what the nanny sees in their table
type NannyPaymentView struct {
	Amount    float64   `json:"amount"` // The net_amount (90%)
	Month     string    `json:"month"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"date"`
}

// PaymentPaginationRequest handles search, sort, and filters
type PaymentPaginationRequest struct {
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=10"`
	Search string `form:"search"` // Search by JobRef or MpesaReceipt
	SortBy string `form:"sort_by,default=created_at"`
	Order  string `form:"order,default=desc"`
	Month  int    `form:"month"`
	Year   int    `form:"year"`
	Status string `form:"status"`
}
type MpesaWebhookInput struct {
	TransactionType   string  `json:"TransactionType"`
	TransID           string  `json:"TransID"` // M-Pesa Receipt (e.g., RKT8S9J2KP)
	TransTime         string  `json:"TransTime"`
	TransAmount       float64 `json:"TransAmount"`       // The cash sent
	BusinessShortCode string  `json:"BusinessShortCode"` // Your Paybill 880100
	BillRefNumber     string  `json:"BillRefNumber"`     // The JobRef (The Account No)
	InvoiceNumber     string  `json:"InvoiceNumber"`
	OrgAccountBalance float64 `json:"OrgAccountBalance"`
	MSISDN            string  `json:"MSISDN"` // Payer phone number
	FirstName         string  `json:"FirstName"`
}

type STKPushRequest struct {
	PhoneNumber string `json:"phone_number"` // This tag MUST match Postman exactly
	JobRef      string `json:"job_ref"`      // This tag MUST match Postman exactly
}

// Used ONLY when the employer makes a typo on their phone.
type ManualPaymentInput struct {
	JobRef          string  `json:"job_ref" binding:"required"`
	Amount          float64 `json:"amount" binding:"required"`
	ExternalReceipt string  `json:"external_receipt" binding:"required"`
	Notes           string  `json:"notes"` // Reason for the manual fix
}
