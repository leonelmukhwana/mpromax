package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/repository"
)

type IncidentService interface {
	CreateIncident(ctx context.Context, reporterID uuid.UUID, reporterRole string, req dto.CreateIncidentRequest) error
	GetAdminReportDashboard(ctx context.Context, params dto.IncidentFilterParams) (dto.IncidentListResponse, error)
	UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string, notes string) error
	GetUserReports(ctx context.Context, userID uuid.UUID) ([]dto.UserIncidentResponse, error)
}

type incidentService struct {
	repo      repository.IncidentRepository
	nannyRepo repository.NannyRepository // Used to verify roles if needed
}

func NewIncidentService(repo repository.IncidentRepository, nannyRepo repository.NannyRepository) IncidentService {
	return &incidentService{
		repo:      repo,
		nannyRepo: nannyRepo,
	}
}

// CreateIncident implements the "Secure Handshake" business logic
func (s *incidentService) CreateIncident(ctx context.Context, reporterID uuid.UUID, reporterRole string, req dto.CreateIncidentRequest) error {

	// 1. SECURITY HANDSHAKE
	// Verify that the reporter and the reported party actually have a shared assignment.
	isValid, err := s.repo.ValidateAssignmentHandshake(ctx, req.AssignmentID, reporterID, req.ReportedID)
	if err != nil {
		return fmt.Errorf("failed to verify assignment: %w", err)
	}

	if !isValid {
		return errors.New("security violation: no valid assignment found between these parties")
	}

	// 1.5 SPAM GATEKEEPER (New Cooldown Logic)
	// Prevent duplicate reports for the same assignment that are still being processed.
	exists, err := s.repo.HasActiveReport(ctx, reporterID, req.AssignmentID)
	if err != nil {
		return fmt.Errorf("failed to check existing reports: %w", err)
	}

	// UPDATED MESSAGE:
	if exists {
		return errors.New("a similar report exists and is not resolved yet")
	}

	// 2. DATA PREPARATION
	incident := &models.IncidentReport{
		ID:           uuid.New(),
		AssignmentID: req.AssignmentID,
		ReporterID:   reporterID,
		ReporterRole: reporterRole,
		ReportedID:   req.ReportedID,
		Subject:      req.Subject,
		Description:  req.Description,
		Status:       "pending",
	}

	// 3. PERSISTENCE
	return s.repo.Create(ctx, incident)
}

// GetAdminReportDashboard handles the administrative overview with search/filter/sort
func (s *incidentService) GetAdminReportDashboard(ctx context.Context, params dto.IncidentFilterParams) (dto.IncidentListResponse, error) {

	// 1. INPUT VALIDATION & SANITIZATION
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 15 // Default safe limit
	}

	// 2. REPOSITORY CALL
	reports, err := s.repo.GetAdminReports(ctx, params)
	if err != nil {
		return dto.IncidentListResponse{}, fmt.Errorf("error retrieving admin reports: %w", err)
	}

	// 3. RESPONSE MAPPING
	// We wrap the result in a ListResponse DTO to support future metadata (like total counts)
	return dto.IncidentListResponse{
		Data:  reports,
		Total: len(reports),
	}, nil
}

// change status of the report
func (s *incidentService) UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string, notes string) error {
	// 1. Update the status in the DB
	err := s.repo.UpdateIncidentStatus(ctx, id, status, notes)
	if err != nil {
		return err
	}

	// 2. TODO: Trigger a notification to the reporter that their status has changed
	// s.worker.EnqueueNotification(reporterID, "Your report status is now " + status)

	return nil
}

func (s *incidentService) GetUserReports(ctx context.Context, userID uuid.UUID) ([]dto.UserIncidentResponse, error) {
	return s.repo.GetUserReports(ctx, userID)
}
