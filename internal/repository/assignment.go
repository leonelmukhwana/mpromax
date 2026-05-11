package repository

import (
	"context"
	"fmt"
	"time"

	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssignmentRepository interface {
	Create(ctx context.Context, a *models.NannyAssignment) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.NannyAssignment, error)
	GetAll(ctx context.Context, filter map[string]interface{}) ([]models.NannyAssignment, error)
	Count(ctx context.Context, filter map[string]interface{}) (int, error)

	IsNannyEligible(ctx context.Context, nannyID uuid.UUID) (bool, error)
	GetJobSnapshot(ctx context.Context, jobID uuid.UUID) (*models.NannyAssignment, error)
}

type assignmentRepo struct {
	db *pgxpool.Pool
}

func NewAssignmentRepository(db *pgxpool.Pool) AssignmentRepository {
	return &assignmentRepo{db: db}
}

// ✅ CREATE
// Fixed: Changed client_id to employer_id and added ID to the insert
func (r *assignmentRepo) Create(ctx context.Context, a *models.NannyAssignment) error {
	query := `
	INSERT INTO nanny_assignments (
		id, job_id, nanny_id, employer_id,
		job_ref, county, residence,
		salary_amount, duration_months,
		status, assignment_date
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		a.ID,
		a.JobID,
		a.NannyID,
		a.EmployerID, // This is the User UUID
		a.JobRef,
		a.County,
		a.Residence,
		a.SalaryAmount,
		a.DurationMonths,
		a.Status,
		time.Now(),
	)

	return err
}

// ✅ ELIGIBILITY CHECK
func (r *assignmentRepo) IsNannyEligible(ctx context.Context, nannyID uuid.UUID) (bool, error) {
	query := `
		SELECT 
			COALESCE(u.profile_complete, FALSE),
			COALESCE(v.is_verification_complete, FALSE)
		FROM users u
		LEFT JOIN nanny_verifications v ON v.nanny_id = u.id 
		WHERE u.id = $1
	`

	var profileComplete bool
	var verified bool

	err := r.db.QueryRow(ctx, query, nannyID).Scan(
		&profileComplete,
		&verified,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return profileComplete && verified, nil
}

// ✅ JOB SNAPSHOT
func (r *assignmentRepo) GetJobSnapshot(ctx context.Context, jobID uuid.UUID) (*models.NannyAssignment, error) {
	query := `
		SELECT 
			j.id,
			j.user_id, 
			j.job_ref,
			j.county,
			j.residence,
			j.salary_amount,
			j.duration_months
		FROM jobs j
		INNER JOIN client_profiles cp ON j.user_id = cp.user_id
		WHERE j.id = $1
	`

	var a models.NannyAssignment
	err := r.db.QueryRow(ctx, query, jobID).Scan(
		&a.JobID,
		&a.EmployerID,
		&a.JobRef,
		&a.County,
		&a.Residence,
		&a.SalaryAmount,
		&a.DurationMonths,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job not found or employer profile missing for ID: %s", jobID)
		}
		return nil, fmt.Errorf("failed to fetch job snapshot: %v", err)
	}

	return &a, nil
}

// ✅ GET BY ID (With Joined Names)
func (r *assignmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.NannyAssignment, error) {
	query := `
        SELECT 
            na.id, na.job_id, na.nanny_id, na.employer_id, na.job_ref, 
            na.county, na.residence, na.salary_amount, na.duration_months, 
            na.status, na.assignment_date, na.created_at, na.updated_at,
            COALESCE(np.full_name, 'Unknown Nanny') as nanny_name,
            COALESCE(cp.full_name, 'Unknown Employer') as employer_name
        FROM nanny_assignments na
        -- Join profiles to get the names
        LEFT JOIN nanny_profiles np ON na.nanny_id = np.user_id
        LEFT JOIN client_profiles cp ON na.employer_id = cp.user_id
        WHERE na.id = $1`

	var a models.NannyAssignment

	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID,
		&a.JobID,
		&a.NannyID,
		&a.EmployerID,
		&a.JobRef,
		&a.County,
		&a.Residence,
		&a.SalaryAmount,
		&a.DurationMonths,
		&a.Status,
		&a.AssignmentDate,
		&a.CreatedAt,
		&a.UpdatedAt,
		&a.NannyName,    // Scanned from joined nanny_profiles
		&a.EmployerName, // Scanned from joined client_profiles
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, err
	}

	return &a, nil
}

// ✅ FILTERED LIST (With Joined Names)
func (r *assignmentRepo) GetAll(ctx context.Context, filter map[string]interface{}) ([]models.NannyAssignment, error) {
	// We join users to get the base user record,
	// then join profiles to get the full_name field.
	query := `
        SELECT 
            na.id, na.job_id, na.nanny_id, na.employer_id, na.job_ref, 
            na.county, na.residence, na.salary_amount, na.duration_months, 
            na.status, na.assignment_date, na.created_at, na.updated_at,
            COALESCE(np.full_name, 'Unknown Nanny') as nanny_name,
            COALESCE(cp.full_name, 'Unknown Employer') as employer_name
        FROM nanny_assignments na
        -- Join for Nanny Name
        LEFT JOIN nanny_profiles np ON na.nanny_id = np.user_id
        -- Join for Employer Name
        LEFT JOIN client_profiles cp ON na.employer_id = cp.user_id
        WHERE 1=1`

	args := []interface{}{}
	i := 1

	if search, ok := filter["search"].(string); ok && search != "" {
		// Use the table alias 'na' to avoid ambiguity
		query += fmt.Sprintf(" AND na.job_ref ILIKE $%d", i)
		args = append(args, "%"+search+"%")
		i++
	}

	if status, ok := filter["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND na.status = $%d", i)
		args = append(args, status)
		i++
	}

	query += " ORDER BY na.created_at DESC"

	if limit, ok := filter["limit"].(int); ok {
		query += fmt.Sprintf(" LIMIT $%d", i)
		args = append(args, limit)
		i++
	}

	if offset, ok := filter["offset"].(int); ok {
		query += fmt.Sprintf(" OFFSET $%d", i)
		args = append(args, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []models.NannyAssignment{}

	for rows.Next() {
		var a models.NannyAssignment
		err := rows.Scan(
			&a.ID,
			&a.JobID,
			&a.NannyID,
			&a.EmployerID,
			&a.JobRef,
			&a.County,
			&a.Residence,
			&a.SalaryAmount,
			&a.DurationMonths,
			&a.Status,
			&a.AssignmentDate,
			&a.CreatedAt,
			&a.UpdatedAt,
			&a.NannyName,    // Scanned from the LEFT JOIN
			&a.EmployerName, // Scanned from the LEFT JOIN
		)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}

	return list, nil
}

// ✅ COUNT
func (r *assignmentRepo) Count(ctx context.Context, filter map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM nanny_assignments WHERE 1=1`
	args := []interface{}{}
	i := 1

	if search, ok := filter["search"].(string); ok && search != "" {
		query += fmt.Sprintf(" AND job_ref ILIKE $%d", i)
		args = append(args, "%"+search+"%")
		i++
	}

	var total int
	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	return total, err
}
