package repository

import (
	"context"
	"errors"
	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VerificationRepository struct {
	DB *pgxpool.Pool
}

func NewVerificationRepository(db *pgxpool.Pool) *VerificationRepository {
	return &VerificationRepository{DB: db}
}

// CheckEligibility checks if the user is a nanny and has completed their profile
func (r *VerificationRepository) CheckEligibility(ctx context.Context, userID uuid.UUID) (bool, error) {
	var isComplete bool
	var role string

	query := `SELECT profile_complete, role FROM users WHERE id = $1`

	// pgx uses QueryRow(ctx, query, args...)
	err := r.DB.QueryRow(ctx, query, userID).Scan(&isComplete, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, errors.New("user not found")
		}
		return false, err
	}

	if role != "nanny" {
		return false, errors.New("access denied: only nannies can perform this action")
	}

	return isComplete, nil
}

// UpsertVerification saves or updates the verification record
func (r *VerificationRepository) UpsertVerification(ctx context.Context, nannyID uuid.UUID, idUrl, selfieUrl, idPubID, selfiePubID string) error {
	query := `
		INSERT INTO nanny_verifications (
			nanny_id, 
			id_card_url, 
			selfie_url, 
			id_cloudinary_public_id, 
			selfie_cloudinary_public_id, 
			is_verification_complete,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, TRUE, NOW())
		ON CONFLICT (nanny_id) DO UPDATE SET 
			id_card_url = EXCLUDED.id_card_url, 
			selfie_url = EXCLUDED.selfie_url,
			id_cloudinary_public_id = EXCLUDED.id_cloudinary_public_id,
			selfie_cloudinary_public_id = EXCLUDED.selfie_cloudinary_public_id,
			is_verification_complete = TRUE,
			updated_at = NOW()`

	// pgx uses Exec(ctx, query, args...)
	_, err := r.DB.Exec(ctx, query, nannyID, idUrl, selfieUrl, idPubID, selfiePubID)
	return err
}

// GetByNannyID fetches the verification record
func (r *VerificationRepository) GetByNannyID(ctx context.Context, nannyID uuid.UUID) (models.NannyVerification, error) {
	var v models.NannyVerification
	query := `
        SELECT id, nanny_id, id_card_url, selfie_url, id_cloudinary_public_id, 
               selfie_cloudinary_public_id, is_verification_complete 
        FROM nanny_verifications 
        WHERE nanny_id = $1`

	//WRONG: err := r.DB.QueryRow(ctx, query, nannyID).Scan(...)
	// CORRECT: List every single field that matches the SELECT statement
	err := r.DB.QueryRow(ctx, query, nannyID).Scan(
		&v.ID,
		&v.NannyID,
		&v.IDCardURL,
		&v.SelfieURL,
		&v.IDPublicID,
		&v.SelfiePublicID,
		&v.IsVerificationComplete,
	)
	return v, err
}

// GetAllVerifications for Admin dashboard
// Change the return type from []any to []models.NannyVerification
func (r *VerificationRepository) GetAllVerifications(ctx context.Context) ([]models.NannyVerification, error) {
	query := `
		SELECT id, nanny_id, id_card_url, selfie_url, id_cloudinary_public_id, selfie_cloudinary_public_id, is_verification_complete 
		FROM nanny_verifications 
		ORDER BY created_at DESC`

	rows, err := r.DB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.NannyVerification // 👈 Use the actual model type
	for rows.Next() {
		var v models.NannyVerification
		err := rows.Scan(
			&v.ID, &v.NannyID, &v.IDCardURL, &v.SelfieURL, &v.IDPublicID, &v.SelfiePublicID, &v.IsVerificationComplete,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, v)
	}
	return results, nil
}

// AdminArchiveUpdate replaces URLs with placeholder after archival
func (r *VerificationRepository) AdminArchiveUpdate(ctx context.Context, nannyID uuid.UUID) error {
	query := `
		UPDATE nanny_verifications 
		SET id_card_url = 'ARCHIVED', 
		    selfie_url = 'ARCHIVED', 
		    updated_at = NOW() 
		WHERE nanny_id = $1`

	_, err := r.DB.Exec(ctx, query, nannyID)
	return err
}
