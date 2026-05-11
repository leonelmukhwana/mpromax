package services

import (
	"context"
	"fmt"
	"math"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type PaymentService struct {
	repo         *repository.PaymentRepository
	notifService NotificationService
	mpesaService *MpesaService // 1. ADD THIS FIELD
}

// 2. UPDATE CONSTRUCTOR to accept MpesaService
func NewPaymentService(repo *repository.PaymentRepository, notif NotificationService, mpesa *MpesaService) *PaymentService {
	return &PaymentService{
		repo:         repo,
		notifService: notif,
		mpesaService: mpesa,
	}
}

// 1. THE WEBHOOK HANDLER: Processes all M-Pesa (STK Push AND Manual Paybill)
func (s *PaymentService) ProcessMpesaPayment(ctx context.Context, input dto.MpesaWebhookInput) error {
	// A. INTEGRITY CHECK: Find the contract using the JobRef (Account No)
	assignment, err := s.repo.GetByJobRef(ctx, input.BillRefNumber)
	if err != nil {
		return fmt.Errorf("SECURITY ALERT: Payment for unknown JobRef %s: %w", input.BillRefNumber, err)
	}

	// B. AMOUNT VALIDATION
	expectedAmount := assignment.SalaryAmount
	paidAmount := input.TransAmount

	status := "completed"
	if math.Abs(paidAmount-expectedAmount) > 0.1 {
		status = "suspense"
	}

	// C. FINANCIAL SPLIT
	commission := paidAmount * 0.10
	netAmount := paidAmount - commission

	// D. 30-DAY IDEMPOTENCY
	now := time.Now()
	idempotencyKey := fmt.Sprintf("%s-%d-%d", input.BillRefNumber, int(now.Month()), now.Year())

	// E. PREPARE DATA
	payment := &models.Payment{
		ID:               uuid.New(),
		AssignmentID:     assignment.ID,
		JobRef:           input.BillRefNumber,
		MpesaReceipt:     input.TransID,
		TotalAmount:      paidAmount,
		CommissionAmount: commission,
		NetAmount:        netAmount,
		Status:           status,
		BillingMonth:     int(now.Month()),
		BillingYear:      now.Year(),
		IdempotencyKey:   idempotencyKey,
	}

	// F. ATOMIC COMMIT
	err = s.repo.ProcessValidatedPayment(ctx, payment)
	if err != nil {
		return err
	}

	// TRIGGER NOTIFICATION: Notify Employer of M-Pesa Payment
	s.sendPaymentNotification(ctx, assignment.EmployerID, paidAmount, input.TransID)

	return nil
}

// 2. UNIFIED FETCHER
func (s *PaymentService) GetPaymentsForUser(ctx context.Context, userID string, role string, req dto.PaymentPaginationRequest) (interface{}, int64, error) {
	switch role {
	case "admin":
		return s.repo.GetAdminPayments(ctx, req)
	case "employer":
		return s.repo.GetEmployerPayments(ctx, userID, req)
	case "nanny":
		return s.repo.GetNannyPayments(ctx, userID, req)
	default:
		return nil, 0, fmt.Errorf("unauthorized role: %s", role)
	}
}

// 3. INITIATE STK
func (s *PaymentService) InitiateSTK(ctx context.Context, userID string, req dto.STKPushRequest) error {
	// 1. Fetch the assignment from the database to get the real amount
	assignment, err := s.repo.GetByJobRef(ctx, req.JobRef)
	if err != nil {
		return fmt.Errorf("invalid job reference: %w", err)
	}

	// 2. IMPORTANT: Check if the amount is valid
	if assignment.SalaryAmount <= 0 {
		return fmt.Errorf("cannot initiate payment for KES 0.00")
	}

	// 3. THE MISSING LINK: Actually trigger the M-Pesa Service
	// This line sends the request to Safaricom's servers
	fmt.Printf("DEBUG: Handing off to M-Pesa API for Phone: %s, Amount: %.2f\n", req.PhoneNumber, assignment.SalaryAmount)

	return s.mpesaService.InitiateSTK(
		req.PhoneNumber,
		int(assignment.SalaryAmount),
		req.JobRef,
	)
}

// 4. MANUAL PROCESS
func (s *PaymentService) ManualProcess(ctx context.Context, input dto.ManualPaymentInput) error {
	// 1. Find the assignment
	assignment, err := s.repo.GetByJobRef(ctx, input.JobRef)
	if err != nil {
		return fmt.Errorf("cannot reconcile: JobRef %s not found", input.JobRef)
	}

	// 2. Apply the standard 10/90 split
	commission := input.Amount * 0.10
	netAmount := input.Amount - commission

	now := time.Now()
	idempotencyKey := fmt.Sprintf("MANUAL-%s-%s", assignment.ID.String(), input.ExternalReceipt)

	payment := &models.Payment{
		ID:               uuid.New(),
		AssignmentID:     assignment.ID,
		JobRef:           input.JobRef,
		MpesaReceipt:     input.ExternalReceipt,
		TotalAmount:      input.Amount,
		CommissionAmount: commission,
		NetAmount:        netAmount,
		Status:           "completed",
		BillingMonth:     int(now.Month()),
		BillingYear:      now.Year(),
		IdempotencyKey:   idempotencyKey,
	}

	err = s.repo.ProcessValidatedPayment(ctx, payment)
	if err != nil {
		return err
	}

	// TRIGGER NOTIFICATION: Notify Employer of Manual Payment
	s.sendPaymentNotification(ctx, assignment.EmployerID, input.Amount, input.ExternalReceipt)

	return nil
}

// Private helper to handle the notification logic
func (s *PaymentService) sendPaymentNotification(ctx context.Context, employerID uuid.UUID, amount float64, receipt string) {
	req := models.NotificationRequest{
		UserID:    employerID,
		EventType: "payment_received",
		Channels:  []string{"email", "push", "web"},
		Payload: models.NotificationPayload{
			Title: "Payment Received! 💳",
			Body:  fmt.Sprintf("Confirmed! We have received KES %.2f (Receipt: %s). Thank you for your payment.", amount, receipt),
		},
	}
	_ = s.notifService.Send(ctx, req)
}
