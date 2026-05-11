package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"rest_api/internal/dto"
	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

// ProcessValidatedPayment implements Atomic Transaction: Payment + Ledger + Outbox
func (r *PaymentRepository) ProcessValidatedPayment(ctx context.Context, p *models.Payment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Insert Payment Record
	paymentQuery := `
		INSERT INTO payments (
			id, assignment_id, job_ref, mpesa_receipt, total_amount, 
			commission_amount, net_amount, status, billing_month, 
			billing_year, idempotency_key, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`

	_, err = tx.Exec(ctx, paymentQuery,
		p.ID, p.AssignmentID, p.JobRef, p.MpesaReceipt, p.TotalAmount,
		p.CommissionAmount, p.NetAmount, p.Status, p.BillingMonth,
		p.BillingYear, p.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("payment insert failed: %w", err)
	}

	// 2. Double-Entry Ledger (Stage 4)
	ledgerQuery := `INSERT INTO payment_ledger (id, payment_id, account, credit) VALUES ($1, $2, $3, $4)`

	// Record Platform 10%
	if _, err = tx.Exec(ctx, ledgerQuery, uuid.New(), p.ID, "platform_commission", p.CommissionAmount); err != nil {
		return err
	}
	// Record Nanny 90%
	if _, err = tx.Exec(ctx, ledgerQuery, uuid.New(), p.ID, "nanny_net", p.NetAmount); err != nil {
		return err
	}

	// 3. Outbox Pattern (Stage 5)
	payload, _ := json.Marshal(map[string]interface{}{
		"assignment_id": p.AssignmentID,
		"status":        "paid",
	})

	outboxQuery := `INSERT INTO payment_outbox (id, event_type, payload) VALUES ($1, $2, $3)`
	if _, err = tx.Exec(ctx, outboxQuery, uuid.New(), "assignment.paid", payload); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetByJobRef: Used by Service to find which Job the Paybill 'Account' belongs to
func (r *PaymentRepository) GetByJobRef(ctx context.Context, jobRef string) (*models.AssignmentDetail, error) {
	var ad models.AssignmentDetail

	query := `
		SELECT a.id, a.employer_id, a.nanny_id, j.salary_amount 
		FROM nanny_assignments a
		JOIN jobs j ON a.job_id = j.id
		WHERE j.job_ref = $1 LIMIT 1`

	err := r.pool.QueryRow(ctx, query, jobRef).Scan(
		&ad.ID, &ad.EmployerID, &ad.NannyID, &ad.SalaryAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("assignment not found for ref %s: %w", jobRef, err)
	}
	return &ad, nil
}

// GetAdminPayments: Master table for Admin
func (r *PaymentRepository) GetAdminPayments(ctx context.Context, req dto.PaymentPaginationRequest) ([]dto.PaymentResponse, int64, error) {
	var results []dto.PaymentResponse
	var total int64

	searchTerm := "%" + req.Search + "%"

	// 1. Accurate Count Query
	countQuery := `
        SELECT COUNT(*) 
        FROM payments p
        WHERE ($1 = '%%' OR p.job_ref ILIKE $1 OR p.mpesa_receipt ILIKE $1)`

	err := r.pool.QueryRow(ctx, countQuery, searchTerm).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count error: %v", err)
	}

	// 2. Updated Join Logic: Linking to the profile tables
	query := `
        SELECT p.id, p.assignment_id, p.job_ref, p.mpesa_receipt, 
               cp.full_name AS employer_name, 
               np.full_name AS nanny_name,
               p.total_amount, p.commission_amount, p.net_amount, 
               p.status, p.billing_month, p.billing_year, p.created_at
        FROM payments p
        JOIN nanny_assignments na ON p.assignment_id = na.id
        JOIN client_profiles cp ON na.employer_id = cp.user_id
        JOIN nanny_profiles np ON na.nanny_id = np.user_id
        WHERE ($1 = '%%' OR p.job_ref ILIKE $1 OR p.mpesa_receipt ILIKE $1)
        ORDER BY p.created_at DESC 
        LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, searchTerm, req.Limit, (req.Page-1)*req.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var res dto.PaymentResponse
		var month int

		err := rows.Scan(
			&res.ID, &res.AssignmentID, &res.JobRef, &res.MpesaReceipt,
			&res.EmployerName, &res.NannyName, &res.TotalAmount, &res.CommissionAmount,
			&res.NetAmount, &res.Status, &month, &res.BillingYear, &res.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan error: %v", err)
		}

		res.BillingMonth = fmt.Sprintf("%d", month)
		results = append(results, res)
	}

	if results == nil {
		results = []dto.PaymentResponse{}
	}

	return results, total, nil
}

// GetEmployerPayments: 100% Salary View
func (r *PaymentRepository) GetEmployerPayments(ctx context.Context, employerID string, req dto.PaymentPaginationRequest) ([]dto.EmployerPaymentView, int64, error) {
	var results []dto.EmployerPaymentView
	var total int64

	// 1. Count remains the same
	countQ := `SELECT COUNT(*) FROM payments p JOIN nanny_assignments na ON p.assignment_id = na.id WHERE na.employer_id = $1`
	r.pool.QueryRow(ctx, countQ, employerID).Scan(&total)

	// 2. Main Query: Swapping n_user for nanny_profiles (np)
	query := `
        SELECT np.full_name, p.total_amount, p.billing_month, p.status, p.mpesa_receipt, p.created_at
        FROM payments p
        JOIN nanny_assignments na ON p.assignment_id = na.id
        JOIN nanny_profiles np ON na.nanny_id = np.user_id
        WHERE na.employer_id = $1
        ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, employerID, req.Limit, (req.Page-1)*req.Limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var v dto.EmployerPaymentView
		var m int
		// Scanning np.full_name into v.NannyName
		err := rows.Scan(&v.NannyName, &v.AmountPaid, &m, &v.Status, &v.MpesaReceipt, &v.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		v.Month = fmt.Sprintf("%d", m)
		results = append(results, v)
	}

	if results == nil {
		results = []dto.EmployerPaymentView{}
	}

	return results, total, nil
}

// GetNannyPayments: 90% Net Earnings View
func (r *PaymentRepository) GetNannyPayments(ctx context.Context, nannyID string, req dto.PaymentPaginationRequest) ([]dto.NannyPaymentView, int64, error) {
	var results []dto.NannyPaymentView
	var total int64

	// 1. Count
	countQ := `SELECT COUNT(*) FROM payments p JOIN nanny_assignments na ON p.assignment_id = na.id WHERE na.nanny_id = $1`
	r.pool.QueryRow(ctx, countQ, nannyID).Scan(&total)

	// 2. Query (Note: month is scanned into a temp variable 'm' first)
	query := `
		SELECT p.net_amount, p.billing_month, p.status, p.created_at
		FROM payments p
		JOIN nanny_assignments na ON p.assignment_id = na.id
		WHERE na.nanny_id = $1
		ORDER BY p.created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, nannyID, req.Limit, (req.Page-1)*req.Limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var v dto.NannyPaymentView
		var m int

		// Map the database columns to your struct fields
		err := rows.Scan(
			&v.Amount,    // p.net_amount
			&m,           // p.billing_month
			&v.Status,    // p.status
			&v.CreatedAt, // p.created_at
		)
		if err != nil {
			return nil, 0, err
		}

		// Convert the integer month to string as per your struct
		v.Month = fmt.Sprintf("%d", m)
		results = append(results, v)
	}

	if results == nil {
		results = []dto.NannyPaymentView{}
	}

	return results, total, nil
}

func (r *PaymentRepository) VerifyJobRef(ctx context.Context, jobRef string) (bool, float64, error) {
	var salary float64
	var id string

	// Query to check if the job_ref exists and get the salary
	query := `SELECT id, salary FROM nanny_assignments WHERE job_ref = $1 LIMIT 1`

	err := r.pool.QueryRow(ctx, query, jobRef).Scan(&id, &salary)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No assignment found with that Job Ref
			return false, 0, nil
		}
		// A real database error occurred
		return false, 0, err
	}

	// Success: Job Ref exists, return the expected salary
	return true, salary, nil
}
