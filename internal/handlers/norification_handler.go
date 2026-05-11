package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"rest_api/internal/services"
)

type NotificationHandler struct {
	notifService services.NotificationService
}

func NewNotificationHandler(s services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: s}
}

// SendManualNotification handles a custom request to notify a user (Admin Only)
// POST /api/v1/admin/notifications/dispatch
func (h *NotificationHandler) SendManualNotification(c *gin.Context) {
	var req struct {
		UserID    uuid.UUID              `json:"user_id" binding:"required"`
		EventType string                 `json:"event_type" binding:"required"`
		Channels  []string               `json:"channels" binding:"required"`
		Payload   map[string]interface{} `json:"payload" binding:"required"`
		ExpiryMin int                    `json:"expiry_minutes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	var expiryPtr *time.Duration
	if req.ExpiryMin > 0 {
		d := time.Duration(req.ExpiryMin) * time.Minute
		expiryPtr = &d
	}

	err := h.notifService.Dispatch(c.Request.Context(), req.UserID, req.EventType, req.Channels, req.Payload, expiryPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue notification"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "Notification queued"})
}

// GetMyHistory allows a user to see their personal notification history (Inbox)
// GET /api/v1/notifications
func (h *NotificationHandler) GetMyHistory(c *gin.Context) {
	// Retrieve UserID from the Auth Middleware
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := val.(uuid.UUID)

	history, err := h.notifService.GetHistoryForUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch notification history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetOutboxStatus allows Admins to monitor the delivery queue
// GET /api/v1/admin/notifications/outbox
func (h *NotificationHandler) GetOutboxStatus(c *gin.Context) {
	// Optional filter: /outbox?status=failed
	statusFilter := c.Query("status")

	outbox, err := h.notifService.GetAllOutboxEntries(c.Request.Context(), statusFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch outbox status"})
		return
	}

	c.JSON(http.StatusOK, outbox)
}
