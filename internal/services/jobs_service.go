package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"rest_api/internal/core/utils"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"
)

type JobService struct {
	Repo *repository.JobRepository
	DB   *pgxpool.Pool
}

func NewJobService(db *pgxpool.Pool, repo *repository.JobRepository) *JobService {
	return &JobService{
		DB:   db,
		Repo: repo,
	}
}

// 1. Create Job
// Fixed: Now uses the UserID directly to match the new database schema.
func (s *JobService) CreateJob(ctx context.Context, userID uuid.UUID, d *dto.CreateJobDTO) (*models.Job, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Generate JobRef loop
	var jobRef string
	for i := 0; i < 10; i++ {
		ref, _ := utils.GenerateJobRef()
		exists, err := s.Repo.ExistsByRef(ctx, tx, ref)
		if err != nil {
			return nil, err
		}
		if !exists {
			jobRef = ref
			break
		}
	}

	// 2. Create the Job struct
	// CRITICAL FIX: We no longer call GetClientIDByUserID.
	// We use the userID from the context directly because jobs.user_id
	// now references users.id (Foreign Key).
	newJob := &models.Job{
		ID:             uuid.New(),
		UserID:         userID, // Direct User UUID
		JobRef:         jobRef,
		EngagementType: d.EngagementType,
		DurationMonths: d.DurationMonths,
		SalaryAmount:   d.SalaryAmount,
		Description:    d.Description,
		County:         d.County,
		Residence:      d.Residence,
		Requirements:   d.Requirements,
		Status:         "open",
	}

	// 3. Save to Repository
	if err := s.Repo.Create(ctx, tx, newJob); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return newJob, nil
}

// 2. Update Job
func (s *JobService) UpdateJob(ctx context.Context, userID uuid.UUID, jobID uuid.UUID, d *dto.UpdateJobDTO) (*models.Job, error) {
	// The repo handles the salary floor check and user_id ownership check
	err := s.Repo.Update(ctx, userID, jobID, d)
	if err != nil {
		return nil, err
	}

	// Return the updated job
	return s.Repo.GetByID(ctx, jobID, userID)
}

// 3. Delete Job
func (s *JobService) DeleteJob(ctx context.Context, jobID, userID uuid.UUID) error {
	return s.Repo.Delete(ctx, jobID, userID)
}

// 4. Admin: Paginated Search/List
func (s *JobService) AdminListJobs(ctx context.Context, f *dto.JobFilterDTO) ([]models.Job, int, error) {
	f.SetDefaults()
	return s.Repo.AdminListJobs(ctx, f)
}

// 5. View Single Job
func (s *JobService) GetJobForUser(ctx context.Context, jobID uuid.UUID, userID uuid.UUID) (*models.Job, error) {
	job, err := s.Repo.GetByID(ctx, jobID, userID)
	if err != nil {
		return nil, err
	}

	return job, nil
}
