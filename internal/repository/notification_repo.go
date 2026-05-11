package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"rest_api/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 1. THE INTERFACE (This is what services look for)
type NotificationRepository interface {
	CreateOutboxEntry(ctx context.Context, n *models.NotificationOutbox) error
	GetPendingOutbox(ctx context.Context) ([]*models.NotificationOutbox, error)
	UpdateOutboxStatus(ctx context.Context, id uuid.UUID, emailStat, pushStat, webStat, errLog string, retryCount int) error
	CreateHistory(ctx context.Context, n *models.Notification) error
	GetOutboxLogs(ctx context.Context, status string) ([]*models.NotificationOutbox, error)
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]*models.Notification, error)
}

// 2. THE STRUCT (Internal implementation)
type notificationRepository struct {
	pool *pgxpool.Pool
}

// 3. THE CONSTRUCTOR (Now correctly returns the Interface)
func NewNotificationRepository(pool *pgxpool.Pool) NotificationRepository {
	return &notificationRepository{pool: pool}
}

// --- OUTBOX LOGIC ---

func (r *notificationRepository) CreateOutboxEntry(ctx context.Context, n *models.NotificationOutbox) error {
	query := `
        INSERT INTO notification_outbox (
            user_id, event_type, payload, email_status, push_status, web_status, 
            retry_count, expires_at, created_at, next_retry_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        RETURNING id, created_at
    `
	payloadJSON, _ := json.Marshal(n.Payload)

	return r.pool.QueryRow(ctx, query,
		n.UserID,
		n.EventType,
		payloadJSON,
		n.EmailStatus,
		n.PushStatus,
		n.WebStatus,
		0,
		n.ExpiresAt,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *notificationRepository) GetPendingOutbox(ctx context.Context) ([]*models.NotificationOutbox, error) {
	query := `
        SELECT id, user_id, event_type, payload, email_status, push_status, web_status, retry_count
        FROM notification_outbox
        WHERE (email_status = 'pending' OR push_status = 'pending' OR web_status = 'pending')
          AND retry_count < 3
          AND (next_retry_at IS NULL OR next_retry_at <= NOW())
        LIMIT 10
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.NotificationOutbox
	for rows.Next() {
		t := &models.NotificationOutbox{}

		// Scan directly into the t.Payload (if it's a map or struct)
		// OR scan into a variable that matches the DB type
		err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.EventType,
			&t.Payload, // pgx usually handles scanning map/struct to JSONB directly
			&t.EmailStatus,
			&t.PushStatus,
			&t.WebStatus,
			&t.RetryCount,
		)
		if err != nil {
			log.Printf("❌ Scan Error in Repository: %v", err) // Add this log!
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}
func (r *notificationRepository) UpdateOutboxStatus(ctx context.Context, id uuid.UUID, emailStat, pushStat, webStat, errLog string, retryCount int) error {
	query := `
        UPDATE notification_outbox 
        SET email_status = $1, 
            push_status = $2, 
            web_status = $3, 
            last_error = $4,
            retry_count = $5,
            next_retry_at = CASE WHEN $5 < 3 THEN NOW() + (interval '1 minute' * $5 * $5) ELSE next_retry_at END
        WHERE id = $6
    `
	_, err := r.pool.Exec(ctx, query, emailStat, pushStat, webStat, errLog, retryCount, id)
	return err
}

// --- HISTORY LOGIC ---

func (r *notificationRepository) CreateHistory(ctx context.Context, n *models.Notification) error {
	query := `
        INSERT INTO notifications (user_id, title, message, channel, type, status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
    `
	_, err := r.pool.Exec(ctx, query, n.UserID, n.Title, n.Message, n.Channel, n.Type, "sent")
	return err
}

func (r *notificationRepository) GetOutboxLogs(ctx context.Context, status string) ([]*models.NotificationOutbox, error) {
	query := `
        SELECT id, user_id, event_type, payload, email_status, push_status, web_status, retry_count, last_error, created_at
        FROM notification_outbox
        WHERE ($1 = '' OR email_status = $1 OR push_status = $1 OR web_status = $1)
        ORDER BY created_at DESC
        LIMIT 100
    `
	rows, err := r.pool.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.NotificationOutbox
	for rows.Next() {
		l := &models.NotificationOutbox{}
		var payloadBytes []byte
		err := rows.Scan(&l.ID, &l.UserID, &l.EventType, &payloadBytes, &l.EmailStatus, &l.PushStatus, &l.WebStatus, &l.RetryCount, &l.LastError, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(payloadBytes, &l.Payload)
		logs = append(logs, l)
	}
	return logs, nil
}

func (r *notificationRepository) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit int) ([]*models.Notification, error) {
	query := `
        SELECT id, user_id, title, message, channel, type, status, metadata, retries, created_at
        FROM notifications
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2
    `
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying notifications failed: %w", err)
	}
	defer rows.Close()

	var notifications []*models.Notification
	for rows.Next() {
		n := &models.Notification{}
		var metadataBytes []byte
		err := rows.Scan(
			&n.ID, &n.UserID, &n.Title, &n.Message, &n.Channel, &n.Type, &n.Status, &metadataBytes, &n.Retries, &n.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning notification row failed: %w", err)
		}

		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &n.Metadata)
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}
