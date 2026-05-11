package repository

import (
	"context"
	"errors"

	"rest_api/internal/dto"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContractRepository defines the menu of operations for handling legal documents.
type ContractRepository interface {
	GetAssignmentDataForContract(ctx context.Context, assignmentID uuid.UUID) (*dto.ContractSourceData, error)
	UpdateAssignmentContractURLs(ctx context.Context, assignmentID uuid.UUID, employerURL, nannyURL string) error
	GetContractURL(ctx context.Context, assignmentID uuid.UUID, userID uuid.UUID) (string, error)
}

// contractRepository implements the ContractRepository interface.
type contractRepository struct {
	db *pgxpool.Pool
}

// NewContractRepository creates a new instance of the repository.
func NewContractRepository(db *pgxpool.Pool) ContractRepository {
	return &contractRepository{db: db}
}

// GetAssignmentDataForContract pulls all raw data across 4 tables for the PDF engine.
// This is used for the automatic generation trigger.
func (r *contractRepository) GetAssignmentDataForContract(ctx context.Context, assignmentID uuid.UUID) (*dto.ContractSourceData, error) {
	query := `
		SELECT 
			a.job_ref, 
			a.salary, 
			a.duration_months, 
			a.start_date, 
			a.residence, 
			a.county,
			cp.fullname as employer_name, 
			cp.id_number as employer_id, 
			cp.phone as employer_phone, 
			u_emp.email as employer_email,
			np.fullname as nanny_name, 
			np.id_number as nanny_id_no, 
			np.phone as nanny_phone
		FROM nanny_assignments a
		JOIN client_profiles cp ON a.client_id = cp.user_id
		JOIN users u_emp ON cp.user_id = u_emp.id
		JOIN nanny_profiles np ON a.nanny_id = np.user_id
		WHERE a.id = $1`

	var d dto.ContractSourceData
	err := r.db.QueryRow(ctx, query, assignmentID).Scan(
		&d.JobRef,
		&d.Salary,
		&d.DurationMonths,
		&d.StartDate,
		&d.Residence,
		&d.County,
		&d.EmployerName,
		&d.EmployerIDNo,
		&d.EmployerPhone,
		&d.EmployerEmail,
		&d.NannyName,
		&d.NannyIDNo,
		&d.NannyPhone,
	)

	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateAssignmentContractURLs saves the Cloudinary links back to the database.
func (r *contractRepository) UpdateAssignmentContractURLs(ctx context.Context, assignmentID uuid.UUID, empURL, nannyURL string) error {
	query := `
		UPDATE nanny_assignments 
		SET employer_contract_url = $1, nanny_contract_url = $2, updated_at = NOW() 
		WHERE id = $3`

	_, err := r.db.Exec(ctx, query, empURL, nannyURL, assignmentID)
	return err
}

// GetContractURL provides secure access to the PDFs.
// It ensures an Employer cannot see the Nanny's contract and vice versa.
func (r *contractRepository) GetContractURL(ctx context.Context, assignmentID uuid.UUID, userID uuid.UUID) (string, error) {
	var clientID, nannyID uuid.UUID
	var empURL, nanURL string

	query := `
		SELECT client_id, nanny_id, employer_contract_url, nanny_contract_url 
		FROM nanny_assignments 
		WHERE id = $1`

	err := r.db.QueryRow(ctx, query, assignmentID).Scan(&clientID, &nannyID, &empURL, &nanURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", errors.New("assignment record not found")
		}
		return "", err
	}

	// PRIVACY FIREWALL: Compare logged-in userID from JWT with DB record
	if userID == clientID {
		if empURL == "" {
			return "", errors.New("employer contract has not been generated yet")
		}
		return empURL, nil
	}

	if userID == nannyID {
		if nanURL == "" {
			return "", errors.New("nanny deployment order has not been generated yet")
		}
		return nanURL, nil
	}

	// If the user is logged in but isn't the client or the nanny for THIS assignment
	return "", errors.New("unauthorized: you do not have permission to view this document")
}
