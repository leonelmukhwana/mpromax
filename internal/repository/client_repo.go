package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rest_api/internal/models"
)

// ClientRepository defines the behavior for client data operations.
// This interface allows main.go to pass this repo to multiple services.
type ClientRepository interface {
	Create(ctx context.Context, tx pgx.Tx, c *models.ClientProfile) error
	ExistsByPhoneHash(ctx context.Context, tx pgx.Tx, hash string) (bool, error)
	GetByUserID(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*models.ClientProfile, error)
	GetByUserIDIncludingDeleted(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (*models.ClientProfile, error)
	UpdateFields(ctx context.Context, tx pgx.Tx, userID uuid.UUID, fields map[string]any) error
	SoftDelete(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error
	Recover(ctx context.Context, tx pgx.Tx, userID uuid.UUID) error
	AdminGetByID(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*models.ClientProfile, error)
	AdminList(ctx context.Context, tx pgx.Tx, limit int, cursor *time.Time, search string, county *string, gender *string) ([]models.ClientProfile, error)
}

// clientRepositoryImpl is the concrete implementation of the interface.
type clientRepositoryImpl struct {
	db *pgxpool.Pool
}

// NewClientRepository now returns the ClientRepository INTERFACE.
// This fixes the "cannot use as value" error in main.go.
func NewClientRepository(db *pgxpool.Pool) ClientRepository {
	return &clientRepositoryImpl{db: db}
}

// Create: Inserts a new client profile
func (r *clientRepositoryImpl) Create(
	ctx context.Context,
	tx pgx.Tx,
	c *models.ClientProfile,
) error {
	query := `
    INSERT INTO client_profiles (
        id, user_id, full_name,
        id_number_encrypted, id_number_hash,
        passport_number_encrypted, passport_number_hash,
        phone_encrypted, phone_hash,
        gender, nationality, county, residence
    )
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
    `

	_, err := tx.Exec(ctx, query,
		c.ID,
		c.UserID,
		c.FullName,
		c.IDNumberEncrypted,
		c.IDNumberHash,
		c.PassportNumberEncrypted,
		c.PassportNumberHash,
		c.PhoneEncrypted,
		c.PhoneHash,
		c.Gender,
		c.Nationality,
		c.County,
		c.Residence,
	)

	return err
}

// ExistsByPhoneHash: Checks for existing phone numbers (active profiles only)
func (r *clientRepositoryImpl) ExistsByPhoneHash(
	ctx context.Context,
	tx pgx.Tx,
	hash string,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (
            SELECT 1 FROM client_profiles WHERE phone_hash=$1 AND is_deleted=FALSE
        )`,
		hash,
	).Scan(&exists)

	return exists, err
}

// GetByUserID: Standard getter for active profiles
func (r *clientRepositoryImpl) GetByUserID(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) (*models.ClientProfile, error) {
	query := `
    SELECT 
        id, user_id, full_name,
        id_number_encrypted, passport_number_encrypted,
        phone_encrypted,
        gender, nationality, county, residence,
        created_at
    FROM client_profiles
    WHERE user_id=$1 AND is_deleted=FALSE
    `

	var c models.ClientProfile

	err := tx.QueryRow(ctx, query, userID).Scan(
		&c.ID,
		&c.UserID,
		&c.FullName,
		&c.IDNumberEncrypted,
		&c.PassportNumberEncrypted,
		&c.PhoneEncrypted,
		&c.Gender,
		&c.Nationality,
		&c.County,
		&c.Residence,
		&c.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// GetByUserIDIncludingDeleted: FIXES "no rows in result set" during recovery
func (r *clientRepositoryImpl) GetByUserIDIncludingDeleted(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) (*models.ClientProfile, error) {
	query := `
    SELECT 
        id, user_id, is_deleted, deleted_at,
        full_name, phone_encrypted, gender, 
        nationality, county, residence, created_at
    FROM client_profiles
    WHERE user_id=$1
    `

	var c models.ClientProfile

	err := tx.QueryRow(ctx, query, userID).Scan(
		&c.ID,
		&c.UserID,
		&c.IsDeleted,
		&c.DeletedAt,
		&c.FullName,
		&c.PhoneEncrypted,
		&c.Gender,
		&c.Nationality,
		&c.County,
		&c.Residence,
		&c.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// UpdateFields: Updates only specific fields provided in the map
func (r *clientRepositoryImpl) UpdateFields(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}

	query := `UPDATE client_profiles SET `
	args := []any{}
	i := 1

	for k, v := range fields {
		query += k + "=$" + strconv.Itoa(i) + ", "
		args = append(args, v)
		i++
	}

	query += "updated_at=NOW() WHERE user_id=$" + strconv.Itoa(i) + " AND is_deleted=FALSE"
	args = append(args, userID)

	_, err := tx.Exec(ctx, query, args...)
	return err
}

// SoftDelete: Marks profile as deleted without removing row
func (r *clientRepositoryImpl) SoftDelete(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) error {
	_, err := tx.Exec(ctx, `
    UPDATE client_profiles
    SET is_deleted=TRUE, deleted_at=NOW()
    WHERE user_id=$1 AND is_deleted=FALSE
    `, userID)

	return err
}

// Recover: Reverses soft delete
func (r *clientRepositoryImpl) Recover(
	ctx context.Context,
	tx pgx.Tx,
	userID uuid.UUID,
) error {
	_, err := tx.Exec(ctx, `
    UPDATE client_profiles
    SET is_deleted=FALSE, deleted_at=NULL
    WHERE user_id=$1 AND is_deleted=TRUE
    `, userID)

	return err
}

// AdminGetByID: Detailed view for admins
func (r *clientRepositoryImpl) AdminGetByID(
	ctx context.Context,
	tx pgx.Tx,
	id uuid.UUID,
) (*models.ClientProfile, error) {
	query := `
    SELECT 
        id, full_name, gender,
        nationality, county, residence,
        id_number_encrypted,
        passport_number_encrypted,
        phone_encrypted,
        created_at,
        is_deleted,
        deleted_at
    FROM client_profiles
    WHERE id=$1
    `

	var c models.ClientProfile

	err := tx.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.FullName,
		&c.Gender,
		&c.Nationality,
		&c.County,
		&c.Residence,
		&c.IDNumberEncrypted,
		&c.PassportNumberEncrypted,
		&c.PhoneEncrypted,
		&c.CreatedAt,
		&c.IsDeleted,
		&c.DeletedAt,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// AdminList: Lists active clients with filtering and cursor pagination
func (r *clientRepositoryImpl) AdminList(
	ctx context.Context,
	tx pgx.Tx,
	limit int,
	cursor *time.Time,
	search string,
	county *string,
	gender *string,
) ([]models.ClientProfile, error) {
	query := `
    SELECT 
        id, full_name, gender,
        nationality, county, residence,
        phone_encrypted, id_number_encrypted, passport_number_encrypted,
        created_at
    FROM client_profiles
    WHERE is_deleted=FALSE
    `

	args := []any{}
	i := 1

	if search != "" {
		query += ` AND full_name ILIKE $` + strconv.Itoa(i)
		args = append(args, "%"+search+"%")
		i++
	}

	if county != nil && *county != "" {
		query += ` AND county=$` + strconv.Itoa(i)
		args = append(args, *county)
		i++
	}

	if gender != nil && *gender != "" {
		query += ` AND gender=$` + strconv.Itoa(i)
		args = append(args, *gender)
		i++
	}

	if cursor != nil {
		query += ` AND created_at < $` + strconv.Itoa(i)
		args = append(args, *cursor)
		i++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(i)
	args = append(args, limit)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.ClientProfile

	for rows.Next() {
		var c models.ClientProfile
		err := rows.Scan(
			&c.ID,
			&c.FullName,
			&c.Gender,
			&c.Nationality,
			&c.County,
			&c.Residence,
			&c.PhoneEncrypted,
			&c.IDNumberEncrypted,
			&c.PassportNumberEncrypted,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}

	return result, nil
}
