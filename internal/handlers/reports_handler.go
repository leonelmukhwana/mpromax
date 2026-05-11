package handlers

import (
	"fmt"
	"net/http"

	"rest_api/internal/dto"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IncidentHandler struct {
	service services.IncidentService
}

func NewIncidentHandler(s services.IncidentService) *IncidentHandler {
	return &IncidentHandler{service: s}
}

// SubmitReport handles POST /api/v1/incidents
// SubmitReport handles the "One-Way" submission for Nannies and Employers
func (h *IncidentHandler) SubmitReport(c *gin.Context) {
	var req dto.CreateIncidentRequest

	// 1. Bind JSON Input
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields or invalid format"})
		return
	}

	// 2. Extract Identity from Auth Middleware
	uid, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User identity not found"})
		return
	}
	userID := uid.(uuid.UUID)

	role, _ := c.Get("user_role")
	userRole := role.(string)

	// 3. Call Service Layer
	err := h.service.CreateIncident(c.Request.Context(), userID, userRole, req)
	if err != nil {
		// Log the actual error for the developer console
		fmt.Printf("DEBUG: Incident failed for User %s: %v\n", userID, err)

		// Handle specific business logic errors
		if err.Error() == "a similar report exists and is not resolved yet" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		// Handle security violations
		if err.Error() == "security violation: no valid assignment found between these parties" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		// Default fallback for unexpected errors (Database down, etc.)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to process report. Ensure the assignment ID is correct."})
		return
	}

	// 4. Success Response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Report submitted successfully. Administration will review the incident.",
	})
}

// AdminListReports handles GET /api/v1/admin/incidents
// Only accessible via Admin Middleware
func (h *IncidentHandler) AdminListReports(c *gin.Context) {
	var params dto.IncidentFilterParams

	// 1. Bind Query Parameters (?status=pending&search=mary)
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filter or search parameters"})
		return
	}

	// 2. Call Service
	response, err := h.service.GetAdminReportDashboard(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve incident logs"})
		return
	}

	// 3. Return JSON with Total Count for Admin Pagination
	c.JSON(http.StatusOK, response)
}

// PATCH /admin/incidents/:id/status
func (h *IncidentHandler) UpdateIncidentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID"})
		return
	}

	var req dto.UpdateIncidentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.UpdateIncidentStatus(c.Request.Context(), id, req.Status, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update incident"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// GET /incidents/my-reports
func (h *IncidentHandler) GetMyReports(c *gin.Context) {
	// Retrieve the userID from the Gin Context (set by your JWT middleware)
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context not found"})
		return
	}

	userID := val.(uuid.UUID)

	reports, err := h.service.GetUserReports(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve your reports"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reports,
		"total": len(reports),
	})
}
