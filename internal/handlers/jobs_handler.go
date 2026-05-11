package handlers

import (
	"net/http"

	"rest_api/internal/dto"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobHandler struct {
	Service *services.JobService
}

func NewJobHandler(service *services.JobService) *JobHandler {
	return &JobHandler{Service: service}
}

// 1. POST /jobs - Employer creates a job
func (h *JobHandler) Create(c *gin.Context) {
	var input dto.CreateJobDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Try all common context keys
	var val any
	keys := []string{"userID", "user_id", "id", "sub"}
	for _, key := range keys {
		if v, exists := c.Get(key); exists {
			val = v
			break
		}
	}

	if val == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: identity not found in context"})
		return
	}

	// 2. Safely parse into a UUID
	var employerID uuid.UUID
	switch v := val.(type) {
	case uuid.UUID:
		employerID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid identity format"})
			return
		}
		employerID = parsed
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error: unexpected id type"})
		return
	}

	// 3. Proceed to Service
	job, err := h.Service.CreateJob(c.Request.Context(), employerID, &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

// 2. PATCH /jobs/:id - Employer updates a job
func (h *JobHandler) Update(c *gin.Context) {
	// 1. Extract the Job ID from the URL path
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id format"})
		return
	}

	// 2. Bind the incoming JSON to the Update DTO
	var input dto.UpdateJobDTO
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Get the User ID from the context (set by your Auth middleware)
	val, exists := c.Get("userID")
	if !exists {
		// Fallback check if your middleware uses "user_id"
		val, exists = c.Get("user_id")
	}

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	userID := val.(uuid.UUID)

	// 4. Call the Service
	// Since the Repo now handles the salary check, the Service will
	// return that error directly to 'err' here.
	job, err := h.Service.UpdateJob(c.Request.Context(), userID, jobID, &input)
	if err != nil {
		// This captures "amount can not be lower" OR "job not found"
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Success! Return the updated job object
	c.JSON(http.StatusOK, job)
}

// 3. DELETE /jobs/:id - Employer deletes an unassigned job
func (h *JobHandler) Delete(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	userID, _ := c.Get("userID")
	employerID := userID.(uuid.UUID)

	err = h.Service.DeleteJob(c.Request.Context(), jobID, employerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job deleted successfully"})
}

// 4. GET /admin/jobs - Admin paginated search and filter
func (h *JobHandler) AdminList(c *gin.Context) {
	var filter dto.JobFilterDTO

	// We use ShouldBindQuery to extract ?page=1&page_size=10
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🛡️ Ensure we have valid numbers before hitting the DB
	filter.SetDefaults()

	jobs, total, err := h.Service.AdminListJobs(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      jobs,
		"total":     total,
		"page":      filter.Page,
		"page_size": filter.PageSize,
	})
}

// 5. GET /jobs/:id - View single job details
func (h *JobHandler) GetJob(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id format"})
		return
	}

	// 1. FIX: Use "user_id" to match your debug output
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Ensure you are asserting to the correct type (uuid.UUID or string)
	// Based on the debug, if it's already a UUID object:
	userID := val.(uuid.UUID)

	// 2. OPTIONAL: Since you have user_role in the context,
	// you can pass it to the service if you want to handle logic there
	// role, _ := c.Get("user_role")

	job, err := h.Service.GetJobForUser(c.Request.Context(), jobID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, job)
}
