package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"rest_api/internal/models"
	"rest_api/internal/repository"
)

type NotificationService interface {
	Dispatch(ctx context.Context, userID uuid.UUID, eventType string, channels []string, payload map[string]interface{}, expiry *time.Duration) error
	GetHistoryForUser(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error)
	GetAllOutboxEntries(ctx context.Context, status string) ([]*models.NotificationOutbox, error)
	Send(ctx context.Context, req models.NotificationRequest) error
}

type notificationService struct {
	// Use the interface types here, not pointers to the struct
	repo       repository.NotificationRepository
	authRepo   repository.AuthRepository
	nannyRepo  repository.NannyRepository
	clientRepo repository.ClientRepository
}

func NewNotificationService(
	repo repository.NotificationRepository,
	authRepo repository.AuthRepository,
	nannyRepo repository.NannyRepository,
	clientRepo repository.ClientRepository,
) NotificationService {
	return &notificationService{
		repo:       repo,
		authRepo:   authRepo,
		nannyRepo:  nannyRepo,
		clientRepo: clientRepo,
	}
}

// --- THIS WAS THE MISSING METHOD CAUSING THE ERROR ---
func (s *notificationService) Send(ctx context.Context, req models.NotificationRequest) error {
	payload := make(map[string]interface{})
	payload["title"] = req.Payload.Title
	payload["message"] = req.Payload.Body // Worker expects "message"

	// 1. Fetch user from the main users table (Works for Admin, Nanny, and Client)
	user, err := s.authRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return fmt.Errorf("could not find user for notification: %w", err)
	}

	// 2. Add the email to the payload so the worker knows where to send it
	payload["email"] = user.Email

	// 3. Copy metadata
	for k, v := range req.Payload.Metadata {
		payload[k] = v
	}

	return s.Dispatch(ctx, req.UserID, req.EventType, req.Channels, payload, nil)
}

// --- EXISTING DISPATCH LOGIC ---
func (s *notificationService) Dispatch(ctx context.Context, userID uuid.UUID, eventType string, channels []string, payload map[string]interface{}, expiry *time.Duration) error {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	emailStatus := "skipped"
	pushStatus := "skipped"
	webStatus := "skipped"

	for _, ch := range channels {
		switch ch {
		case "web":
			webStatus = "pending"
		case "email":
			emailStatus = "pending"
		case "push":
			token, _ := s.getFCMTokenForUser(ctx, userID)
			if token != "" {
				payload["fcm_token"] = token
				pushStatus = "pending"
			} else {
				log.Printf("⚠️  Skipping Push for %s: No FCM Token", userID)
				pushStatus = "skipped"
			}
		}
	}

	var expiresAt *time.Time
	if expiry != nil {
		t := time.Now().Add(*expiry)
		expiresAt = &t
	}

	outboxEntry := &models.NotificationOutbox{
		UserID:      userID,
		EventType:   eventType,
		Payload:     payload,
		EmailStatus: emailStatus,
		PushStatus:  pushStatus,
		WebStatus:   webStatus,
		ExpiresAt:   expiresAt,
	}

	return s.repo.CreateOutboxEntry(ctx, outboxEntry)
}

// --- OTHER METHODS ---

func (s *notificationService) GetHistoryForUser(ctx context.Context, userID uuid.UUID) ([]*models.Notification, error) {
	return s.repo.GetUserNotifications(ctx, userID, 50)
}

func (s *notificationService) GetAllOutboxEntries(ctx context.Context, status string) ([]*models.NotificationOutbox, error) {
	return s.repo.GetOutboxLogs(ctx, status)
}

func (s *notificationService) getFCMTokenForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	nanny, err := s.nannyRepo.GetByUserID(ctx, userID)
	if err == nil && nanny != nil && nanny.FCMToken != "" {
		return nanny.FCMToken, nil
	}

	client, err := s.clientRepo.GetByUserID(ctx, nil, userID)
	if err == nil && client != nil && client.FCMToken != "" {
		return client.FCMToken, nil
	}

	return "", fmt.Errorf("token not found")
}
