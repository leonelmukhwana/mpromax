package handlers

import (
	"net/http"
	"rest_api/internal/dto"
	"rest_api/internal/models"
	"rest_api/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NannyHandler struct {
	service services.NannyService
}

func NewNannyHandler(s services.NannyService) *NannyHandler {
	return &NannyHandler{service: s}
}

// CREATE: POST /profiles/nanny
func (h *NannyHandler) Create(c *gin.Context) {
	var req dto.NannyProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.service.CreateNannyProfile(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Profile created successfully"})
}

// VIEW OWN: GET /profiles/nanny/me
func (h *NannyHandler) GetMe(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	profile, err := h.service.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// UPDATE OWN: PATCH /profiles/nanny/me
func (h *NannyHandler) UpdateMe(c *gin.Context) {
	var req dto.NannyUpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.service.UpdateNannyProfile(c.Request.Context(), userID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profile updated"})
}

// --- ADMIN HANDLERS ---

// LIST: GET /admin/nannies
func (h *NannyHandler) AdminList(c *gin.Context) {
	var filter models.NannySearchFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.AdminListNannies(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch nannies"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE: DELETE /admin/nannies/:id
func (h *NannyHandler) AdminDelete(c *gin.Context) {
	nannyID, _ := uuid.Parse(c.Param("id"))
	actorID := c.MustGet("user_id").(uuid.UUID)

	var req dto.NannyDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.DeleteNanny(c.Request.Context(), nannyID, actorID, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profile soft-deleted"})
}

// RECOVER: POST /admin/nannies/:id/recover
func (h *NannyHandler) AdminRecover(c *gin.Context) {
	nannyID, _ := uuid.Parse(c.Param("id"))
	actorID := c.MustGet("user_id").(uuid.UUID)

	if err := h.service.RecoverNanny(c.Request.Context(), nannyID, actorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Profile recovered"})
}

// AdminGet: GET /api/v1/admin/nannies/:id
// This allows the admin to view a specific nanny's full details,
// including the decrypted ID and Phone number.
func (h *NannyHandler) AdminGet(c *gin.Context) {
	// 1. Parse the ID from the URL parameter (:id)
	idParam := c.Param("id")
	nannyID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Nanny ID format"})
		return
	}

	// 2. Call the service to get the decrypted data
	// The service already handles the GetProfileByID call from the repo
	profile, err := h.service.AdminGetNanny(c.Request.Context(), nannyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nanny profile not found or has been permanently deleted"})
		return
	}

	// 3. Return the NannyAdminViewResponse DTO
	c.JSON(http.StatusOK, profile)
}
