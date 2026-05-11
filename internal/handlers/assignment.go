package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"rest_api/internal/dto"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AssignmentHandler struct {
	svc services.AssignmentService
}

func NewAssignmentHandler(s services.AssignmentService) *AssignmentHandler {
	return &AssignmentHandler{svc: s}
}

// ✅ Create
func (h *AssignmentHandler) Create(c *gin.Context) {
	var req dto.CreateAssignmentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.svc.CreateAssignment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "assignment created"})
}

// ✅ Admin List
func (h *AssignmentHandler) GetAll(c *gin.Context) {

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	filter := dto.AssignmentFilter{
		Search:   c.Query("search"),
		Status:   c.Query("status"),
		Page:     page,
		PageSize: pageSize,
	}

	data, total, err := h.svc.GetAssignments(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  data,
		"total": total,
	})
}

// ✅ Nanny View (NO SALARY)
// ✅ Nanny View (Salary Hidden via DTO)
func (h *AssignmentHandler) GetForNanny(c *gin.Context) {
	// 1. Extract User ID safely (handling both string and UUID types)
	var loggedInNannyID string
	if val, exists := c.Get("user_id"); exists {
		loggedInNannyID = fmt.Sprintf("%v", val)
	}

	// 2. Parse the Assignment ID from the URL
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assignment id format"})
		return
	}

	// 3. Fetch assignment from service
	assignment, err := h.svc.GetAssignmentByID(c.Request.Context(), assignmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
		return
	}

	// 4. Debugging Logs (Check your terminal!)
	fmt.Printf("[DEBUG] Token Nanny ID: %s\n", loggedInNannyID)
	fmt.Printf("[DEBUG] DB Nanny ID:    %s\n", assignment.NannyID.String())

	// 5. Ownership Check: Only allow the assigned nanny to view this
	if assignment.NannyID.String() != loggedInNannyID {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not authorized to view this assignment"})
		return
	}

	// 6. Map to DTO (Excluding Salary for privacy)
	response := dto.NannyAssignmentResponse{
		ID:             assignment.ID,
		JobRef:         assignment.JobRef,
		County:         assignment.County,
		Residence:      assignment.Residence,
		DurationMonths: assignment.DurationMonths,
		Status:         assignment.Status,
		AssignmentDate: assignment.AssignmentDate,
		EmployerName:   assignment.EmployerName,
	}

	c.JSON(http.StatusOK, response)
}
