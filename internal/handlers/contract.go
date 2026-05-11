package handlers

import (
	"net/http"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContractHandler handles incoming HTTP requests related to legal documents.
type ContractHandler struct {
	svc services.ContractService
}

// NewContractHandler creates a new instance of the contract handler.
func NewContractHandler(s services.ContractService) *ContractHandler {
	return &ContractHandler{svc: s}
}

// DownloadContract retrieves the secure Cloudinary URL for a user's contract.
// It relies on the Service and Repository to verify that the requesting user
// is actually a party (Client or Nanny) involved in the assignment.
func (h *ContractHandler) DownloadContract(c *gin.Context) {
	// 1. Extract the Assignment ID
	idParam := c.Param("id")
	assignmentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid assignment ID"})
		return
	}

	// 2. Retrieve User ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "error", "message": "User context not found"})
		return
	}

	// 3. Retrieve User ROLE from context (CRITICAL for the fix)
	userRoleValue, roleExists := c.Get("user_role")
	userRole := ""
	if roleExists {
		userRole = userRoleValue.(string)
	}

	// Convert userID to uuid.UUID
	var userID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		userID, _ = uuid.Parse(v)
	case uuid.UUID:
		userID = v
	}

	// 4. Call Service Layer with the correct 4 arguments
	// Added 'userRole' as the 4th argument here
	downloadURL, err := h.svc.GetUserContract(c.Request.Context(), assignmentID, userID, userRole)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	// 5. Success Response
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"assignment_id": assignmentID,
			"download_url":  downloadURL,
		},
	})
}

// download
// AdminForceGenerate allows an admin to manually trigger the PDF generation and upload process.
// This is useful if the initial automatic generation failed or if assignment details were updated.
func (h *ContractHandler) AdminForceGenerate(c *gin.Context) {
	// 1. Extract Assignment ID
	idParam := c.Param("id")
	assignmentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid assignment ID format",
		})
		return
	}

	// 2. Call the Service to (re)generate contracts
	// We use the background context or request context
	err = h.svc.AutoGenerateAssignmentContracts(c.Request.Context(), assignmentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to regenerate contracts: " + err.Error(),
		})
		return
	}

	// 3. Success Response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Contracts successfully regenerated and uploaded to Cloudinary",
	})
}
