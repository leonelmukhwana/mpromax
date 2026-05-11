package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"rest_api/internal/dto"
	"rest_api/internal/models"
)

type JobRepository struct {
	DB *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{DB: db}
}

// 1. Create Job
// Now inserts the UserID directly into the jobs table.
func (r *JobRepository) Create(ctx context.Context, tx pgx.Tx, job *models.Job) error {
	query := `
        INSERT INTO jobs (
            id, user_id, job_ref, engagement_type, 
            duration_months, salary_amount, description, 
            county, residence, requirements, status
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := tx.Exec(ctx, query,
		job.ID,
		job.UserID, // This is now the direct User UUID
		job.JobRef,
		job.EngagementType,
		job.DurationMonths,
		job.SalaryAmount,
		job.Description,
		job.County,
		job.Residence,
		job.Requirements,
		job.Status,
	)
	return err
}

// 2. Check if Ref Exists
func (r *JobRepository) ExistsByRef(ctx context.Context, tx pgx.Tx, ref string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM jobs WHERE job_ref = $1)`
	err := tx.QueryRow(ctx, query, ref).Scan(&exists)
	return exists, err
}

// 3. Update Job
// Simplified to compare user_id directly.
func (r *JobRepository) Update(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, d *dto.UpdateJobDTO) error {
	// Safety Guard: Check salary
	if d.SalaryAmount != nil {
		var actualSalary float64
		err := r.DB.QueryRow(ctx, "SELECT salary_amount FROM jobs WHERE id = $1", jobID).Scan(&actualSalary)
		if err == nil && *d.SalaryAmount < actualSalary {
			return errors.New("amount cannot be lower than existing salary")
		}
	}

	query := `
        UPDATE jobs 
        SET engagement_type = COALESCE($1, engagement_type),
            duration_months = COALESCE($2, duration_months),
            salary_amount   = COALESCE($3, salary_amount),
            description     = COALESCE($4, description),
            county          = COALESCE($5, county),
            residence       = COALESCE($6, residence),
            requirements    = COALESCE($7, requirements),
            updated_at      = NOW()
        WHERE id = $8 
        AND user_id = $9` // Direct match on UserID

	result, err := r.DB.Exec(ctx, query,
		d.EngagementType,
		d.DurationMonths,
		d.SalaryAmount,
		d.Description,
		d.County,
		d.Residence,
		d.Requirements,
		jobID,
		userID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("job not found or unauthorized")
	}

	return nil
}

// 4. Delete Job
// Allows Owner or Admin to delete if the status is 'open'.
func (r *JobRepository) Delete(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) error {
	query := `
        DELETE FROM jobs j
        USING users u
        WHERE j.id = $1
        AND u.id = $2
        AND j.status = 'open'
        AND (j.user_id = $2 OR u.role = 'admin')`

	result, err := r.DB.Exec(ctx, query, jobID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("cannot delete job: must be 'open' and you must be the owner or admin")
	}

	return nil
}

// 5. Admin: Paginated List with Search
func (r *JobRepository) AdminListJobs(ctx context.Context, f *dto.JobFilterDTO) ([]models.Job, int, error) {
	var jobs []models.Job = []models.Job{}
	var totalCount int

	searchQuery := "%" + f.Search + "%"
	offset := (f.Page - 1) * f.PageSize

	countQuery := `SELECT COUNT(*) FROM jobs WHERE (job_ref ILIKE $1 OR description ILIKE $1)`
	err := r.DB.QueryRow(ctx, countQuery, searchQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := `
        SELECT 
            id, user_id, job_ref, engagement_type, duration_months, 
            salary_amount, description, county, residence, requirements, 
            status, created_at
        FROM jobs
        WHERE (job_ref ILIKE $1 OR description ILIKE $1)
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

	rows, err := r.DB.Query(ctx, dataQuery, searchQuery, f.PageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var j models.Job
		err := rows.Scan(
			&j.ID, &j.UserID, &j.JobRef, &j.EngagementType, &j.DurationMonths,
			&j.SalaryAmount, &j.Description, &j.County, &j.Residence, &j.Requirements,
			&j.Status, &j.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}

	return jobs, totalCount, nil
}

// 6. View Single Job
func (r *JobRepository) GetByID(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) (*models.Job, error) {
	var j models.Job

	query := `
        SELECT 
            j.id, j.user_id, j.job_ref, j.engagement_type, 
            j.duration_months, j.salary_amount, j.description, 
            j.county, j.residence, j.requirements, j.status, j.created_at
        FROM jobs j
        JOIN users u ON u.id = $2
        WHERE j.id = $1 
        AND (j.user_id = $2 OR u.role = 'admin')`

	err := r.DB.QueryRow(ctx, query, jobID, userID).Scan(
		&j.ID, &j.UserID, &j.JobRef, &j.EngagementType,
		&j.DurationMonths, &j.SalaryAmount, &j.Description,
		&j.County, &j.Residence, &j.Requirements, &j.Status, &j.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("job not found or access denied")
		}
		return nil, err
	}
	return &j, nil
}

// GetClientIDByUserID is kept for backward compatibility if needed in services,
// but is no longer required for basic Job operations.
func (r *JobRepository) GetClientIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var clientID uuid.UUID
	query := `SELECT id FROM client_profiles WHERE user_id = $1`

	err := r.DB.QueryRow(ctx, query, userID).Scan(&clientID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, errors.New("employer profile not found")
		}
		return uuid.Nil, err
	}
	return clientID, nil
}
