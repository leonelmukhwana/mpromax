package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"rest_api/internal/models"
	"rest_api/internal/pkg/notifications"
	"rest_api/internal/repository"
)

type NotificationWorker struct {
	repo      repository.NotificationRepository
	emailProv *notifications.EmailProvider
	fcmProv   *notifications.FCMProvider
	webProv   *notifications.WebProvider
}

func NewNotificationWorker(
	repo repository.NotificationRepository,
	email *notifications.EmailProvider,
	fcm *notifications.FCMProvider,
	web *notifications.WebProvider,
) *NotificationWorker {
	return &NotificationWorker{
		repo:      repo,
		emailProv: email,
		fcmProv:   fcm,
		webProv:   web,
	}
}

func (w *NotificationWorker) Start(ctx context.Context) {
	log.Println("🚀 Starting Outbox Processor [Event-Driven]...")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOutbox(ctx)
		}
	}
}

func (w *NotificationWorker) processOutbox(ctx context.Context) {
	tasks, err := w.repo.GetPendingOutbox(ctx)
	if err != nil {
		log.Printf("Outbox DB Error: %v", err)
		return
	}

	for _, task := range tasks {
		w.handleTask(ctx, task)
	}
}

func (w *NotificationWorker) handleTask(ctx context.Context, task *models.NotificationOutbox) {
	// We track if anything changes so we only update the DB once per task
	anySuccess := false
	lastError := ""

	// 1. Handle EMAIL
	if task.EmailStatus == "pending" && w.emailProv != nil {
		// In a real app, you'd pull the 'to' address from the payload or user table
		to, _ := task.Payload["email"].(string)
		if to != "" {
			err := w.emailProv.Send(to, "Notification", fmt.Sprintf("%v", task.Payload["message"]))
			if err == nil {
				task.EmailStatus = "sent"
				anySuccess = true
			} else {
				lastError = fmt.Sprintf("Email: %v", err)
			}
		} else {
			task.EmailStatus = "skipped"
		}
	}

	// 2. Handle PUSH
	if task.PushStatus == "pending" && w.fcmProv != nil {
		token, _ := task.Payload["fcm_token"].(string)
		if token != "" {
			title, _ := task.Payload["title"].(string)
			msg, _ := task.Payload["message"].(string)

			err := w.fcmProv.SendPush(ctx, token, title, msg, nil)
			if err == nil {
				task.PushStatus = "sent"
				anySuccess = true
			} else {
				lastError = fmt.Sprintf("Push: %v", err)
			}
		} else {
			task.PushStatus = "skipped"
		}
	}

	// 3. Handle WEB
	if task.WebStatus == "pending" && w.webProv != nil {
		msg, _ := task.Payload["message"].(string)
		err := w.webProv.Broadcast(task.UserID.String(), msg)
		if err == nil {
			task.WebStatus = "sent"
			anySuccess = true
		} else {
			lastError = fmt.Sprintf("Web: %v", err)
		}
	}

	// 4. Update the Outbox Status
	newRetryCount := task.RetryCount
	if !anySuccess && lastError != "" {
		newRetryCount++
	}

	err := w.repo.UpdateOutboxStatus(ctx, task.ID, task.EmailStatus, task.PushStatus, task.WebStatus, lastError, newRetryCount)
	if err != nil {
		log.Printf("Failed to update outbox for %s: %v", task.ID, err)
	}

	// 5. IF SENT SUCCESSFULLY: Create a History Log for the User UI
	if anySuccess {
		w.repo.CreateHistory(ctx, &models.Notification{
			UserID:  task.UserID,
			Title:   fmt.Sprintf("%v", task.Payload["title"]),
			Message: fmt.Sprintf("%v", task.Payload["message"]),
			Channel: "multi",
			Type:    task.EventType,
		})
	}
}
