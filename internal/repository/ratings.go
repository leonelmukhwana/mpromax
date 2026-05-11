package repository

import (
	"context"

	"fmt"
	"time"

	"rest_api/internal/dto"
	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RatingRepository defines the contract for the database operations.
// Adding this fixes the "undefined: RatingRepository" error.
type RatingRepository interface {
	CreateMonthlyRating(ctx context.Context, rating *models.MonthlyRating) error
	GetAdminRatingDashboard(ctx context.Context, filter dto.RatingFilterParams) ([]dto.AdminRatingItem, int64, error)
	GetNannyRatingsPaginated(ctx context.Context, nannyID uuid.UUID, filter dto.RatingFilterParams) ([]dto.NannyRatingItem, int64, error)
}

type ratingRepository struct {
	db *pgxpool.Pool // Primary for Writes
	ro *pgxpool.Pool // Read-Only Replica for Analytics/Dashboards
}

// NewRatingRepository returns the interface type.
func NewRatingRepository(db *pgxpool.Pool, ro *pgxpool.Pool) RatingRepository {
	return &ratingRepository{db: db, ro: ro}
}

// 1. WRITE OPERATION (Transactional + Idempotent)
func (r *ratingRepository) CreateMonthlyRating(ctx context.Context, rating *models.MonthlyRating) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	checkQuery := `
        SELECT EXISTS(
            SELECT 1 FROM nanny_assignments 
            WHERE id = $1 AND employer_id = $2 AND nanny_id = $3 AND status = 'active'
        )`
	if err := tx.QueryRow(ctx, checkQuery, rating.AssignmentID, rating.EmployerID, rating.NannyID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("security_violation: no active assignment found between these parties")
	}

	insertQuery := `
        INSERT INTO nanny_monthly_ratings (
            assignment_id, employer_id, nanny_id, rating_value, 
            review_text, rating_month, rating_year
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (assignment_id, rating_month, rating_year) 
        DO UPDATE SET 
            rating_value = EXCLUDED.rating_value,
            review_text = EXCLUDED.review_text,
            updated_at = NOW()`

	_, err = tx.Exec(ctx, insertQuery,
		rating.AssignmentID, rating.EmployerID, rating.NannyID,
		rating.RatingValue, rating.ReviewText, rating.RatingMonth, rating.RatingYear)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// 2. ADMIN MASTER VIEW (Aggregated + Read Optimized)
// GetAdminRatingDashboard fetches the master view for Admins with aggregated stats and current month details.
func (r *ratingRepository) GetAdminRatingDashboard(ctx context.Context, filter dto.RatingFilterParams) ([]dto.AdminRatingItem, int64, error) {
	offset := (filter.Page - 1) * filter.Limit
	var totalRecords int64

	query := `
        WITH Stats AS (
            SELECT 
                nanny_id,
                SUM(rating_value) as total_acc,
                ROUND((SUM(rating_value)::NUMERIC / (NULLIF(COUNT(id), 0) * 5) * 100), 2) as lifetime_pct
            FROM nanny_monthly_ratings
            GROUP BY nanny_id
        ),
        CurrentView AS (
            SELECT DISTINCT ON (nanny_id)
                nanny_id, rating_value, review_text, rating_month, rating_year
            FROM nanny_monthly_ratings
            WHERE ($1 = 0 OR rating_month = $1) AND ($2 = 0 OR rating_year = $2)
            ORDER BY nanny_id, rating_year DESC, rating_month DESC
        )
        SELECT 
            COALESCE(np.full_name, 'N/A'),
            u_n.email,                                       -- 1. Added Email here
            COALESCE(np.phone_number_encrypted, 'N/A'),
            COALESCE(cp.full_name, 'No Active Assignment'),
            COALESCE(s.total_acc, 0), 
            COALESCE(s.lifetime_pct, 0.0),
            COALESCE(cv.rating_value, 0), 
            COALESCE(cv.review_text, ''), 
            COALESCE(cv.rating_month, 0), 
            COALESCE(cv.rating_year, 0),
            COUNT(*) OVER()
        FROM Stats s
        JOIN CurrentView cv ON s.nanny_id = cv.nanny_id
        JOIN users u_n ON s.nanny_id = u_n.id
        LEFT JOIN nanny_profiles np ON u_n.id = np.user_id
        LEFT JOIN nanny_assignments na ON na.nanny_id = u_n.id AND na.status = 'active'
        LEFT JOIN client_profiles cp ON na.employer_id = cp.user_id
        ORDER BY s.lifetime_pct DESC
        LIMIT $3 OFFSET $4`

	rows, err := r.ro.Query(ctx, query, filter.Month, filter.Year, filter.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []dto.AdminRatingItem
	for rows.Next() {
		var item dto.AdminRatingItem
		var mNum int

		// 2. Updated Scan to include item.NannyEmail (Total 11 items now)
		err := rows.Scan(
			&item.NannyFullName,
			&item.NannyEmail, // Maps to u_n.email
			&item.NannyPhoneNumber,
			&item.EmployerFullName,
			&item.TotalAccumulatedScore,
			&item.LifetimePercentage,
			&item.CurrentMonthRating,
			&item.CurrentMonthReason,
			&mNum,
			&item.CurrentYear,
			&totalRecords,
		)
		if err != nil {
			return nil, 0, err
		}

		if mNum >= 1 && mNum <= 12 {
			item.CurrentMonthName = time.Month(mNum).String()
		} else {
			item.CurrentMonthName = "N/A"
		}

		results = append(results, item)
	}

	return results, totalRecords, nil
}

// 3. NANNY PERSONAL VIEW (Paginated + Filtered)
func (r *ratingRepository) GetNannyRatingsPaginated(ctx context.Context, nannyID uuid.UUID, filter dto.RatingFilterParams) ([]dto.NannyRatingItem, int64, error) {
	offset := (filter.Page - 1) * filter.Limit
	var totalRecords int64

	query := `
        SELECT 
            rating_value, review_text, rating_month, rating_year,
            ROUND((rating_value::NUMERIC / 5 * 100), 2) as month_pct,
            COUNT(*) OVER() as full_count
        FROM nanny_monthly_ratings
        WHERE nanny_id = $1 
        AND ($2 = 0 OR rating_month = $2)
        AND ($3 = 0 OR rating_year = $3)
        ORDER BY rating_year DESC, rating_month DESC
        LIMIT $4 OFFSET $5`

	rows, err := r.ro.Query(ctx, query, nannyID, filter.Month, filter.Year, filter.Limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []dto.NannyRatingItem
	for rows.Next() {
		var i dto.NannyRatingItem
		var mNum int
		if err := rows.Scan(&i.RatingValue, &i.ReviewText, &mNum, &i.Year, &i.MonthlyPerformancePct, &totalRecords); err != nil {
			return nil, 0, err
		}
		i.MonthName = time.Month(mNum).String()
		items = append(items, i)
	}

	return items, totalRecords, nil
}
