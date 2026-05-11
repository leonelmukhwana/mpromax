package handlers

import (
	"net/http"
	"strconv"

	"rest_api/internal/dto"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RatingHandler struct {
	service      services.RatingService
	nannyService services.NannyService // Correctly injected
}

// Updated constructor to accept both services
func NewRatingHandler(s services.RatingService, ns services.NannyService) *RatingHandler {
	return &RatingHandler{
		service:      s,
		nannyService: ns,
	}
}

// 1. SubmitRating: POST /api/v1/ratings
func (h *RatingHandler) SubmitRating(c *gin.Context) {
	var req dto.CreateMonthlyRatingRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}

	req.EmployerID = uid.(uuid.UUID)

	if err := h.service.SubmitMonthlyRating(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Rating saved successfully"})
}

// 2. GetAdminDashboard: GET /api/v1/admin/ratings
func (h *RatingHandler) GetAdminDashboard(c *gin.Context) {
	var params dto.RatingFilterParams

	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters: " + err.Error()})
		return
	}

	response, err := h.service.GetAdminRatingDashboard(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// 3. GetNannyHistory: GET /api/v1/nanny/my-ratings
func (h *RatingHandler) GetNannyHistory(c *gin.Context) {
	// 1. Get the User ID from the token
	val, _ := c.Get("user_id")
	userID := val.(uuid.UUID)

	// 2. Setup Filters
	filter := dto.RatingFilterParams{
		Page:  parseDefaultInt(c.Query("page"), 1),
		Limit: parseDefaultInt(c.Query("limit"), 10),
	}

	// 3. Just call the service.
	// We will update the service to handle the lookup internally.
	response, err := h.service.GetNannyRatingHistory(c.Request.Context(), userID, filter)

	if err != nil {
		// If anything goes wrong, just show "No ratings"
		c.JSON(http.StatusOK, dto.NannyRatingListResponse{
			Data: []dto.NannyRatingItem{},
			Meta: dto.PaginationMeta{TotalRecords: 0},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// --- HELPER UTILS ---

func parseOptionalInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

func parseDefaultInt(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 {
		return defaultVal
	}
	return val
}
