package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"rest_api/internal/core/utils"
	"rest_api/internal/models"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NannyRepository interface {
	CreateProfile(ctx context.Context, p *models.NannyProfile, actorID uuid.UUID) error
	UpdateProfile(ctx context.Context, p *models.NannyProfile, actorID uuid.UUID) error
	GetProfileByID(ctx context.Context, userID uuid.UUID) (*models.NannyProfile, error)
	ListNannies(ctx context.Context, f models.NannySearchFilter) ([]models.NannyProfile, int, error)
	SoftDeleteProfile(ctx context.Context, userID uuid.UUID, actorID uuid.UUID, reason string) error
	RecoverProfile(ctx context.Context, userID uuid.UUID, actorID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.NannyProfile, error)
	GetNannyIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

type nannyRepository struct {
	db *pgxpool.Pool
}

func NewNannyRepository(db *pgxpool.Pool) NannyRepository {
	return &nannyRepository{db: db}
}

func (r *nannyRepository) CreateProfile(ctx context.Context, p *models.NannyProfile, actorID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Encrypt Sensitive Data
	encID, _ := utils.Encrypt(p.IDNumber)
	encPhone, _ := utils.Encrypt(p.PhoneNumber)

	// 2. Insert Profile
	_, err = tx.Exec(ctx, `
        INSERT INTO nanny_profiles (user_id, full_name, id_number_encrypted, phone_number_encrypted, dob, home_county, education_level)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.UserID, p.FullName, encID, encPhone, p.DOB, p.HomeCounty, p.EducationLevel)
	if err != nil {
		return fmt.Errorf("profile insert failed: %w", err)
	}

	// 3. Update User Status
	_, err = tx.Exec(ctx, `UPDATE users SET profile_complete = true, updated_at = NOW() WHERE id = $1`, p.UserID)
	if err != nil {
		return fmt.Errorf("user status update failed: %w", err)
	}

	// 4. Create Audit Log
	newVal, _ := json.Marshal(p)
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_id, new_values) VALUES ($1, 'CREATE_PROFILE', $2, $3)`,
		actorID, p.UserID, newVal)
	if err != nil {
		return fmt.Errorf("audit log failed: %w", err)
	}

	return tx.Commit(ctx)
}

// 2. UPDATE: Transaction for Snapshotting old data + Updating + Audit
func (r *nannyRepository) UpdateProfile(ctx context.Context, p *models.NannyProfile, actorID uuid.UUID) error {
	query := `
        UPDATE nanny_profiles 
        SET full_name = $1, 
            phone_number_encrypted = $2, 
            home_county = $3, 
            education_level = $4,
            version = version + 1  -- Increment the version
        WHERE user_id = $5 
          AND version = $6      -- Only update if version matches what we read
        RETURNING version`

	// Encrypt sensitive data before update
	encPhone, _ := utils.Encrypt(p.PhoneNumber)

	var newVersion int
	err := r.db.QueryRow(ctx, query,
		p.FullName,
		encPhone,
		p.HomeCounty,
		p.EducationLevel,
		p.UserID,
		p.Version, // The version we fetched originally
	).Scan(&newVersion)

	if err != nil {
		if err.Error() == "no rows in result set" {
			// This means the version changed in the background!
			return errors.New("conflict: profile was updated by another user. please refresh")
		}
		return err
	}

	// Update the local object with the new version
	p.Version = newVersion
	return nil
}

// 3. GET SINGLE: Decrypts ID and Phone on the fly
func (r *nannyRepository) GetProfileByID(ctx context.Context, userID uuid.UUID) (*models.NannyProfile, error) {
	var p models.NannyProfile
	var encID, encPhone string

	query := `SELECT user_id, full_name, id_number_encrypted, phone_number_encrypted, dob, home_county, education_level, is_verified, created_at 
	          FROM nanny_profiles WHERE user_id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(ctx, query, userID).Scan(&p.UserID, &p.FullName, &encID, &encPhone, &p.DOB, &p.HomeCounty, &p.EducationLevel, &p.IsVerified, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	p.IDNumber, _ = utils.Decrypt(encID)
	p.PhoneNumber, _ = utils.Decrypt(encPhone)
	p.CalculateAge()
	return &p, nil
}

// 4. LIST: Pagination + Decryption for Admin dashboard
func (r *nannyRepository) ListNannies(ctx context.Context, f models.NannySearchFilter) ([]models.NannyProfile, int, error) {
	var nannies []models.NannyProfile
	var total int
	offset := (f.Page - 1) * f.Limit

	query := `
		SELECT user_id, full_name, id_number_encrypted, phone_number_encrypted, dob, home_county, education_level, created_at, COUNT(*) OVER() 
		FROM nanny_profiles 
		WHERE deleted_at IS NULL 
		AND (full_name ILIKE '%' || $1 || '%' OR home_county ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, f.Name, f.County, f.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.NannyProfile
		var encID, encPhone string
		if err := rows.Scan(&n.UserID, &n.FullName, &encID, &encPhone, &n.DOB, &n.HomeCounty, &n.EducationLevel, &n.CreatedAt, &total); err != nil {
			return nil, 0, err
		}

		n.IDNumber, _ = utils.Decrypt(encID)
		n.PhoneNumber, _ = utils.Decrypt(encPhone)
		n.CalculateAge()
		nannies = append(nannies, n)
	}
	return nannies, total, nil
}

// 5. DELETE: Transaction for Multi-Table Soft Delete
func (r *nannyRepository) SoftDeleteProfile(ctx context.Context, userID uuid.UUID, actorID uuid.UUID, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Mark both User and Profile as deleted so login is disabled immediately
	_, err = tx.Exec(ctx, `UPDATE users SET deleted_at = $1 WHERE id = $2`, now, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE nanny_profiles SET deleted_at = $1 WHERE user_id = $2`, now, userID)
	if err != nil {
		return err
	}

	meta, _ := json.Marshal(map[string]string{"reason": reason})
	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_id, old_values) VALUES ($1, 'SOFT_DELETE', $2, $3)`,
		actorID, userID, meta)

	return tx.Commit(ctx)
}

// 6. RECOVER: Transaction for Restoring Access
func (r *nannyRepository) RecoverProfile(ctx context.Context, userID uuid.UUID, actorID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE users SET deleted_at = NULL WHERE id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE nanny_profiles SET deleted_at = NULL WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_id) VALUES ($1, 'RESTORE_PROFILE', $2)`, actorID, userID)

	return tx.Commit(ctx)
}
func (r *nannyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.NannyProfile, error) {
	p := &models.NannyProfile{}

	// Make sure the SQL column name matches your DB (usually id_card_url)
	query := `
        SELECT user_id, full_name, selfie_url, id_card_url, is_verified 
        FROM nanny_profiles 
        WHERE user_id = $1`

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&p.UserID,
		&p.FullName,
		&p.SelfieURL,
		&p.IDCardURL, // Matches your struct field exactly now
		&p.IsVerified,
	)

	if err != nil {
		return nil, err
	}
	return p, nil
}

// internal/repository/nanny_repository.go

func (r *nannyRepository) GetNannyIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var nannyID uuid.UUID

	// 1. Log the ID being searched so you can verify it against your DB
	log.Printf("DEBUG: Looking for Nanny Profile with UserID: %s", userID.String())

	// 2. Use a more robust query that handles potential type casting issues
	query := `SELECT id FROM nannies WHERE user_id::text = $1::text LIMIT 1`

	err := r.db.QueryRow(ctx, query, userID.String()).Scan(&nannyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Printf("DEBUG: No row found in nannies table for UserID: %s", userID.String())
			return uuid.Nil, fmt.Errorf("nanny_profile_missing")
		}
		return uuid.Nil, err
	}

	log.Printf("DEBUG: Found NannyID: %s", nannyID.String())
	return nannyID, nil
}
