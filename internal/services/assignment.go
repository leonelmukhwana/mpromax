package services

import (
	"context"
	"errors"
	"fmt"

	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"

	"github.com/google/uuid"
)

type AssignmentService interface {
	CreateAssignment(ctx context.Context, req dto.CreateAssignmentRequest) error
	GetAssignments(ctx context.Context, filter dto.AssignmentFilter) ([]models.NannyAssignment, int, error)
	GetAssignmentByID(ctx context.Context, id uuid.UUID) (*models.NannyAssignment, error)
}

type assignmentService struct {
	repo         repository.AssignmentRepository
	contractSvc  ContractService
	notifService NotificationService
}

// Updated Constructor to accept ContractService
func NewAssignmentService(r repository.AssignmentRepository, c ContractService, n NotificationService) AssignmentService {
	return &assignmentService{
		repo:         r,
		contractSvc:  c,
		notifService: n,
	}
}

// ✅ Create Assignment
func (s *assignmentService) CreateAssignment(ctx context.Context, req dto.CreateAssignmentRequest) error {
	// ... (Eligibility and Snapshot checks remain the same) ...
	eligible, err := s.repo.IsNannyEligible(ctx, req.NannyID)
	if err != nil {
		return err
	}
	if !eligible {
		return errors.New("nanny not eligible")
	}

	snapshot, err := s.repo.GetJobSnapshot(ctx, req.JobID)
	if err != nil {
		return err
	}

	assignment := &models.NannyAssignment{
		ID:             uuid.New(),
		JobID:          req.JobID,
		NannyID:        req.NannyID,
		EmployerID:     snapshot.EmployerID,
		JobRef:         snapshot.JobRef,
		County:         snapshot.County,
		Residence:      snapshot.Residence,
		SalaryAmount:   snapshot.SalaryAmount,
		DurationMonths: snapshot.DurationMonths,
		Status:         "active",
	}

	// 1. Save the assignment to the Database
	err = s.repo.Create(ctx, assignment)
	if err != nil {
		return err
	}

	// 2. TRIGGER NOTIFICATIONS (Nanny & Employer)
	// We do this immediately; the Dispatcher will save them to the outbox
	// for the background worker to pick up.

	channels := []string{"email", "push", "web"}

	// Notify Nanny
	s.notifService.Send(ctx, models.NotificationRequest{
		UserID:    assignment.NannyID,
		EventType: "assignment_created",
		Channels:  channels,
		Payload: models.NotificationPayload{
			Title: "New Job Assigned! 💼",
			Body:  fmt.Sprintf("You have been assigned to Job %s in %s.", assignment.JobRef, assignment.County),
		},
	})

	// Notify Employer
	s.notifService.Send(ctx, models.NotificationRequest{
		UserID:    assignment.EmployerID,
		EventType: "nanny_assigned",
		Channels:  channels,
		Payload: models.NotificationPayload{
			Title: "Nanny Assigned! ✅",
			Body:  fmt.Sprintf("A nanny has been successfully assigned to your request (%s).", assignment.JobRef),
		},
	})

	// 3. AUTOMATIC TRIGGER: Generate Contracts
	go func(id uuid.UUID) {
		bgCtx := context.Background()
		// Your existing contract logic...
		_ = s.contractSvc.AutoGenerateAssignmentContracts(bgCtx, id)
	}(assignment.ID)

	return nil
}

// ✅ Get Assignments (Admin)
func (s *assignmentService) GetAssignments(ctx context.Context, f dto.AssignmentFilter) ([]models.NannyAssignment, int, error) {
	filter := map[string]interface{}{
		"search": f.Search,
		"status": f.Status,
		"limit":  f.PageSize,
		"offset": (f.Page - 1) * f.PageSize,
	}

	data, err := s.repo.GetAll(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

// ✅ Get One
func (s *assignmentService) GetAssignmentByID(ctx context.Context, id uuid.UUID) (*models.NannyAssignment, error) {
	return s.repo.GetByID(ctx, id)
}
