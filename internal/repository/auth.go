package repository

import (
	"context"
	"errors"
	"time"

	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 1. THE INTERFACE (The "Menu" of what this repo can do)
// This is what SetupRoutes and Middleware will look for.
type AuthRepository interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	GetByEmail(ctx context.Context, email string) (models.User, error)
	CreateUser(ctx context.Context, u models.User) error
	UpdatePassword(ctx context.Context, email, hash string) error
	UpdateFailedAttempts(ctx context.Context, id uuid.UUID, count int) error
	LockAccount(ctx context.Context, id uuid.UUID) error
	SaveOTP(ctx context.Context, otp models.OTP) error
	VerifyAndExpireOTP(ctx context.Context, userID uuid.UUID, code, otpType string) error
	UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error
	BlacklistToken(ctx context.Context, token string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	ClearExpiredBlacklist(ctx context.Context) error
	UpdateProfileStatus(ctx context.Context, tx pgx.Tx, userID uuid.UUID, status bool) error
	GetPaginatedUsers(ctx context.Context, limit, offset int) ([]models.User, int, error)
	UpdateStatus(ctx context.Context, userID uuid.UUID, newStatus models.Status) error
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
	UpdateEmail(ctx context.Context, userID uuid.UUID, newEmail string) error
}

// 2. THE STRUCT (The "Worker" that holds the DB connection)
// We keep this lowercase (private) so people use the Interface instead.
type authRepository struct {
	db *pgxpool.Pool
}

// 3. THE CONSTRUCTOR
// This returns the INTERFACE type.
func NewAuthRepository(db *pgxpool.Pool) AuthRepository {
	return &authRepository{db: db}
}

// --- ALL METHODS BELOW USE THE (r *authRepository) RECEIVER ---

func (r *authRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

func (r *authRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	query := `SELECT id, email, password_hash, role, status, failed_attempts, locked_until FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Status, &u.FailedAttempts, &u.LockedUntil)
	return u, err
}

func (r *authRepository) CreateUser(ctx context.Context, u models.User) error {
	query := `INSERT INTO users (id, email, password_hash, role, status, created_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query, u.ID, u.Email, u.PasswordHash, u.Role, u.Status, u.CreatedAt)
	return err
}

func (r *authRepository) UpdatePassword(ctx context.Context, email, hash string) error {
	query := `UPDATE users SET password_hash = $1, failed_attempts = 0, locked_until = NULL WHERE email = $2`
	_, err := r.db.Exec(ctx, query, hash, email)
	return err
}

func (r *authRepository) UpdateFailedAttempts(ctx context.Context, id uuid.UUID, count int) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET failed_attempts = $1 WHERE id = $2`, count, id)
	return err
}

func (r *authRepository) LockAccount(ctx context.Context, id uuid.UUID) error {
	until := time.Now().Add(30 * time.Minute)
	_, err := r.db.Exec(ctx, `UPDATE users SET locked_until = $1 WHERE id = $2`, until, id)
	return err
}

func (r *authRepository) SaveOTP(ctx context.Context, otp models.OTP) error {
	query := `INSERT INTO otp_codes (user_id, code, type, expires_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, otp.UserID, otp.Code, otp.Type, otp.ExpiresAt)
	return err
}

func (r *authRepository) VerifyAndExpireOTP(ctx context.Context, userID uuid.UUID, code, otpType string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM otp_codes WHERE user_id = $1 AND code = $2 AND type = $3 AND expires_at > NOW()`, userID, code, otpType)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("invalid or expired code")
	}
	return nil
}

func (r *authRepository) UpdateUserStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *authRepository) BlacklistToken(ctx context.Context, token string, expiresAt time.Time) error {
	query := `INSERT INTO token_blacklist (token, expires_at) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, token, expiresAt)
	return err
}

func (r *authRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM token_blacklist WHERE token = $1)`
	err := r.db.QueryRow(ctx, query, token).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// delete expired tokens
func (r *authRepository) ClearExpiredBlacklist(ctx context.Context) error {
	query := `DELETE FROM token_blacklist WHERE expires_at < NOW()`
	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *authRepository) UpdateProfileStatus(ctx context.Context, tx pgx.Tx, userID uuid.UUID, status bool) error {
	// Changed "is_profile_complete" to "profile_complete"
	query := `UPDATE users SET profile_complete = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(ctx, query, status, userID)
	return err
}

// view list of users
// Fetch all registered users for Admin view
// ✅ IMPLEMENTATION: GetPaginatedUsers
func (r *authRepository) GetPaginatedUsers(ctx context.Context, limit, offset int) ([]models.User, int, error) {
	var total int
	// Get total count for the frontend to calculate pages
	countQuery := `SELECT COUNT(*) FROM users WHERE role != 'admin'`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch the specific page of users
	query := `
        SELECT id, email, role, status, created_at 
        FROM users 
        WHERE role != 'admin'
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.Status, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

// ✅ IMPLEMENTATION: UpdateStatus
func (r *authRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, newStatus models.Status) error {
	query := `UPDATE users SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, newStatus, userID)
	return err
}

func (r *authRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	var user models.User

	// In pgx, the context is the first argument, no .WithContext needed
	query := `SELECT id, email, password_hash, role, status, failed_attempts, locked_until, created_at 
              FROM users WHERE id = $1`

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.FailedAttempts,
		&user.LockedUntil,
		&user.CreatedAt,
	)

	if err != nil {
		return models.User{}, err
	}
	return user, nil
}
func (r *authRepository) UpdateEmail(ctx context.Context, userID uuid.UUID, newEmail string) error {
	query := `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, newEmail, userID)
	return err
}
