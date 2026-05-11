package handlers

import (
	"net/http"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"strconv"

	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BookingHandler struct {
	service services.BookingService
}

func NewBookingHandler(s services.BookingService) *BookingHandler {
	return &BookingHandler{service: s}
}

// Create handles the Nanny booking request
func (h *BookingHandler) Create(c *gin.Context) {
	// 1. Get Nanny ID from Context (Set by AuthMiddleware)
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: User ID missing"})
		return
	}

	var nannyID uuid.UUID

	// Type Assertion: Handle the case where middleware stores it as UUID or string
	switch v := val.(type) {
	case uuid.UUID:
		nannyID = v
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format in token"})
			return
		}
		nannyID = parsedID
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal type mismatch for user_id"})
		return
	}

	// 2. Bind Request Body (DTO)
	var req dto.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Call Service
	// The service handles business logic (8am-5pm, 30-min intervals)
	// The repository handles atomicity and is_verification_complete check
	booking, err := h.service.CreateBooking(c.Request.Context(), nannyID, req.BookingSlot, req.IdempotencyKey)
	if err != nil {
		// Map errors to correct HTTP Status Codes
		status := http.StatusInternalServerError

		switch err.Error() {
		case "cannot book a slot in the past",
			"interviews must be scheduled between 08:00 and 17:00",
			"invalid time slot: please choose a time on the hour or half-hour (e.g., 09:00 or 09:30)":
			status = http.StatusBadRequest
		case "cannot book: nanny verification is incomplete":
			status = http.StatusForbidden // 403
		case "conflict: slot taken, duplicate request, or nanny already booked":
			status = http.StatusConflict // 409
		}

		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// 4. Return Success
	c.JSON(http.StatusCreated, booking)
}

// List handles the Admin search and pagination
// List handles GET /api/v1/admin/bookings (Admin only)
func (h *BookingHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := models.AdminBookingFilter{
		Page:      page,
		Limit:     limit,
		NannyName: c.Query("nanny_name"),
	}

	bookings, total, err := h.service.GetAdminList(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // Show real error for debugging
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookings": bookings, "total": total})
}

// GetMyBookings handles GET /api/v1/profiles/nanny/bookings (Nanny only)
func (h *BookingHandler) GetMyBookings(c *gin.Context) {
	val, _ := c.Get("user_id")
	nannyID := val.(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	bookings, total, err := h.service.GetMyBookings(c.Request.Context(), nannyID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch your bookings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bookings": bookings, "total": total})
}
