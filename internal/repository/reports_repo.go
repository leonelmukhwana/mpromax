package repository

import (
	"context"

	"fmt"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"rest_api/internal/dto"
	"rest_api/internal/models"
)

type IncidentRepository interface {
	Create(ctx context.Context, report *models.IncidentReport) error
	GetAdminReports(ctx context.Context, params dto.IncidentFilterParams) ([]dto.AdminIncidentResponse, error)
	ValidateAssignmentHandshake(ctx context.Context, assignmentID, reporterID, reportedID uuid.UUID) (bool, error)
	UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string, notes string) error
	GetUserReports(ctx context.Context, userID uuid.UUID) ([]dto.UserIncidentResponse, error)
	HasActiveReport(ctx context.Context, reporterID uuid.UUID, assignmentID uuid.UUID) (bool, error)
}

type incidentRepository struct {
	db *pgxpool.Pool
}

func NewIncidentRepository(db *pgxpool.Pool) IncidentRepository {
	return &incidentRepository{db: db}
}

// ValidateAssignmentHandshake ensures a report is only filed between parties of an existing assignment.
// This is a critical security layer.
func (r *incidentRepository) ValidateAssignmentHandshake(ctx context.Context, assignmentID, reporterID, reportedID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM nanny_assignments 
			WHERE id = $1 
			AND (
				(nanny_id = $2 AND employer_id = $3) -- Reporter is Nanny, Reported is Client
				OR 
				(employer_id = $2 AND nanny_id = $3) -- Reporter is Client, Reported is Nanny
			)
		)`

	var exists bool
	err := r.db.QueryRow(ctx, query, assignmentID, reporterID, reportedID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// Create persists a new incident. We return a generic error to the caller to avoid leaking DB details.
func (r *incidentRepository) Create(ctx context.Context, report *models.IncidentReport) error {
	query := `
		INSERT INTO incident_reports (
			assignment_id, reporter_id, reporter_role, reported_id, subject, description, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`

	_, err := r.db.Exec(ctx, query,
		report.AssignmentID,
		report.ReporterID,
		report.ReporterRole,
		report.ReportedID,
		report.Subject,
		report.Description,
		report.Status,
	)
	return err
}

// GetAdminReports applies the Search, Sort, and Filter principles.
func (r *incidentRepository) GetAdminReports(ctx context.Context, params dto.IncidentFilterParams) ([]dto.AdminIncidentResponse, error) {
	var reports []dto.AdminIncidentResponse

	// We use subqueries or CASE statements to pull full_name from the correct profile table
	// depending on whether the user is a nanny or a client.
	query := `
		SELECT 
			ir.id, 
			ir.status, 
			ir.subject, 
			ir.description, 
			-- Get Reporter Name from correct profile
			COALESCE(np1.full_name, cp1.full_name) AS reporter_name,
			u1.email AS reporter_email, 
			ir.reporter_role,
			-- Get Reported Name from correct profile
			COALESCE(np2.full_name, cp2.full_name) AS reported_name,
			u2.email AS reported_email,
			ir.assignment_id, 
			ir.created_at
		FROM incident_reports ir
		-- Joins for Reporter
		INNER JOIN users u1 ON ir.reporter_id = u1.id
		LEFT JOIN nanny_profiles np1 ON u1.id = np1.user_id
		LEFT JOIN client_profiles cp1 ON u1.id = cp1.user_id
		-- Joins for Reported Party
		INNER JOIN users u2 ON ir.reported_id = u2.id
		LEFT JOIN nanny_profiles np2 ON u2.id = np2.user_id
		LEFT JOIN client_profiles cp2 ON u2.id = cp2.user_id
		ORDER BY ir.created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query preparation failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var report dto.AdminIncidentResponse

		// This must match the 11 fields in your DTO perfectly
		err := rows.Scan(
			&report.ID,
			&report.Status,
			&report.Subject,
			&report.Description,
			&report.ReporterName,  // From COALESCE
			&report.ReporterEmail, // From u1.email
			&report.ReporterRole,
			&report.ReportedName,  // From COALESCE
			&report.ReportedEmail, // From u2.email
			&report.AssignmentID,
			&report.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// admin action on status
func (r *incidentRepository) UpdateIncidentStatus(ctx context.Context, id uuid.UUID, status string, notes string) error {
	query := `
		UPDATE incident_reports 
		SET status = $1, admin_notes = $2, updated_at = NOW() 
		WHERE id = $3`

	_, err := r.db.Exec(ctx, query, status, notes, id)
	return err
}

// reporter to sse the proceedigs of his complaint
// Change AdminIncidentResponse -> UserIncidentResponse here
func (r *incidentRepository) GetUserReports(ctx context.Context, userID uuid.UUID) ([]dto.UserIncidentResponse, error) {
	var reports []dto.UserIncidentResponse // Change this line too

	query := `
		SELECT 
			ir.id, ir.status, ir.subject, ir.description, 
			COALESCE(np.full_name, cp.full_name) AS reported_name,
			COALESCE(ir.admin_notes, '') AS admin_notes,
			ir.assignment_id, ir.created_at
		FROM incident_reports ir
		LEFT JOIN nanny_profiles np ON ir.reported_id = np.user_id
		LEFT JOIN client_profiles cp ON ir.reported_id = cp.user_id
		WHERE ir.reporter_id = $1
		ORDER BY ir.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var report dto.UserIncidentResponse // Change this line too
		err := rows.Scan(
			&report.ID,
			&report.Status,
			&report.Subject,
			&report.Description,
			&report.ReportedName,
			&report.AdminNotes,
			&report.AssignmentID,
			&report.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// check for previous report
func (r *incidentRepository) HasActiveReport(ctx context.Context, reporterID uuid.UUID, assignmentID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM incident_reports 
			WHERE reporter_id = $1 
			AND assignment_id = $2 
			AND status IN ('pending', 'investigating')
		)`

	err := r.db.QueryRow(ctx, query, reporterID, assignmentID).Scan(&exists)
	return exists, err
}
