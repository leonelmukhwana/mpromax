package handlers

import (
	"net/http"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VerificationHandler struct {
	Service *services.VerificationService
}

func NewVerificationHandler(svc *services.VerificationService) *VerificationHandler {
	return &VerificationHandler{Service: svc}
}

// Upload handles the Nanny's document submission
func (h *VerificationHandler) Upload(c *gin.Context) {
	// 1. Extract Nanny ID from JWT safely
	// We use a type switch to prevent a panic if the middleware stores a UUID instead of a string
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var nannyID uuid.UUID
	var parseErr error

	switch v := val.(type) {
	case uuid.UUID:
		nannyID = v
	case string:
		nannyID, parseErr = uuid.Parse(v)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Context Error: User ID type mismatch"})
		return
	}

	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format in token"})
		return
	}

	// 2. Get Files from the request
	// Ensure these keys match exactly in Postman's form-data tab
	idHeader, err := c.FormFile("id_card")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID card photo is required (key: id_card)"})
		return
	}

	selfieHeader, err := c.FormFile("selfie")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Selfie photo is required (key: selfie)"})
		return
	}

	// 3. Open file streams
	idFile, err := idHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open ID card file"})
		return
	}
	defer idFile.Close()

	selfieFile, err := selfieHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open selfie file"})
		return
	}
	defer selfieFile.Close()

	// 4. Delegate to Service (Check Profile -> Resize -> Cloudinary -> DB)
	// Safety check: prevent panic if service wasn't initialized in main.go
	if h.Service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Verification service not initialized"})
		return
	}

	err = h.Service.ProcessVerification(c.Request.Context(), nannyID, idFile, selfieFile)
	if err != nil {
		// We return the actual error message here to help you debug (e.g., Cloudinary credentials)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Service failure",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification documents uploaded and status updated successfully",
	})
}

// GetStatus allows the Nanny to check their own verification progress
func (h *VerificationHandler) GetStatus(c *gin.Context) {
	// 1. Safely retrieve the ID from the context
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in session"})
		return
	}

	// 2. Safely convert the ID (handles both string and uuid.UUID types)
	var nannyID uuid.UUID
	switch v := val.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID format"})
			return
		}
		nannyID = parsedID
	case uuid.UUID:
		nannyID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal type mismatch"})
		return
	}

	// 3. Check if Service or Repo is nil before calling
	if h.Service == nil || h.Service.Repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service unavailable"})
		return
	}

	status, err := h.Service.Repo.GetByNannyID(c.Request.Context(), nannyID)
	if err != nil {
		// Log the actual error here for your own debugging
		c.JSON(http.StatusNotFound, gin.H{"error": "Verification record not found"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// AdminPurgeStorage allows the Admin to delete from Cloudinary after local archival
func (h *VerificationHandler) AdminPurgeStorage(c *gin.Context) {
	// Extract the NannyID from URL params (e.g., /admin/verify/purge/:id)
	targetNannyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Nanny ID"})
		return
	}

	err = h.Service.PurgeCloudinaryAssets(c.Request.Context(), targetNannyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to purge storage: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cloudinary space cleared; record preserved as ARCHIVED"})
}

// AdminGetVerifications lists all uploads for the Admin dashboard
func (h *VerificationHandler) AdminGetVerifications(c *gin.Context) {
	// Security Check: The middleware should already ensure the user is an 'admin'
	verifications, err := h.Service.Repo.GetAllVerifications(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch verifications"})
		return
	}

	c.JSON(http.StatusOK, verifications)
}

// AdminGetSingleVerification fetches details for one specific nanny
func (h *VerificationHandler) AdminGetSingleVerification(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Nanny ID"})
		return
	}

	verification, err := h.Service.Repo.GetByNannyID(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, verification)
}
