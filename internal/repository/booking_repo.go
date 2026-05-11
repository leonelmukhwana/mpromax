package repository

import (
	"context"
	"errors"

	"rest_api/internal/dto"
	"rest_api/internal/models"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) CreateBooking(ctx context.Context, b *models.Booking) error {
	// Start Transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // Safe: rollback does nothing if tx is committed

	// 1. ATOMIC CHECK: Is nanny verified?
	var isVerified bool
	queryCheck := `SELECT is_verification_complete FROM nanny_verifications 
               WHERE nanny_id = $1`

	err = tx.QueryRow(ctx, queryCheck, b.NannyID).Scan(&isVerified)
	if err != nil {
		if err == pgx.ErrNoRows {
			return errors.New("nanny verification record not found")
		}
		return err
	}
	if !isVerified {
		return errors.New("cannot book: nanny verification is incomplete")
	}

	// 2. INSERT BOOKING
	// Idempotency and "No double booking" are handled by the UNIQUE constraints in SQL
	queryInsert := `
		INSERT INTO bookings (nanny_id, booking_slot, idempotency_key, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id`

	err = tx.QueryRow(ctx, queryInsert, b.NannyID, b.BookingSlot, b.IdempotencyKey).Scan(&b.ID)
	if err != nil {
		// Handle PostgreSQL Unique Violation (Error Code 23505)
		if strings.Contains(err.Error(), "23505") {
			return errors.New("conflict: slot taken, duplicate request, or nanny already booked")
		}
		return err
	}

	return tx.Commit(ctx)
}

func (r *BookingRepository) GetAdminBookings(ctx context.Context, filter models.AdminBookingFilter) ([]dto.AdminBookingResponse, int64, error) {
	var total int64
	bookings := []dto.AdminBookingResponse{}

	// 1. Updated Join logic (using np.user_id as discussed)
	baseQuery := `
		FROM bookings b
		JOIN users u ON b.nanny_id = u.id
		LEFT JOIN nanny_profiles np ON b.nanny_id = np.user_id
		WHERE b.deleted_at IS NULL`

	// ... (Existing filter logic for NannyName stays here) ...

	// 2. Updated SELECT to include b.status
	selectQuery := `
		SELECT 
			b.id, b.booking_slot, b.created_at, b.status,
			u.id, COALESCE(np.full_name, 'No Name Provided'), u.email
		` + baseQuery + ` 
		ORDER BY b.booking_slot DESC 
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, selectQuery, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var b dto.AdminBookingResponse
		// 3. Scan order must match SELECT order perfectly
		err := rows.Scan(
			&b.ID,
			&b.BookingSlot,
			&b.CreatedAt,
			&b.Status, // Scanned from b.status
			&b.Nanny.ID,
			&b.Nanny.FullName,
			&b.Nanny.Email,
		)
		if err != nil {
			return nil, 0, err
		}

		// 4. Create the "Pretty" date and time string
		b.FormattedSlot = b.BookingSlot.Format("January 02, 2006 - 03:04 PM")

		bookings = append(bookings, b)
	}

	return bookings, total, nil
}

// NANNY BOOKING DETAILS
func (r *BookingRepository) GetNannyBookings(ctx context.Context, nannyID uuid.UUID, limit, offset int) ([]models.Booking, int64, error) {
	var total int64
	var bookings []models.Booking

	// 1. Get total count for this nanny
	countQuery := `SELECT COUNT(*) FROM bookings WHERE nanny_id = $1 AND deleted_at IS NULL`
	if err := r.db.QueryRow(ctx, countQuery, nannyID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 2. Get the slots
	selectQuery := `
        SELECT id, nanny_id, booking_slot, idempotency_key, created_at 
        FROM bookings 
        WHERE nanny_id = $1 AND deleted_at IS NULL
        ORDER BY booking_slot DESC 
        LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, selectQuery, nannyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(&b.ID, &b.NannyID, &b.BookingSlot, &b.IdempotencyKey, &b.CreatedAt); err != nil {
			return nil, 0, err
		}
		bookings = append(bookings, b)
	}

	return bookings, total, nil
}
