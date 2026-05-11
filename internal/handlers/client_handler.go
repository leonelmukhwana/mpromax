package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"rest_api/internal/dto"
	"rest_api/internal/services"
)

type ClientHandler struct {
	Service *services.ClientService
}

func NewClientHandler(service *services.ClientService) *ClientHandler {
	return &ClientHandler{
		Service: service,
	}
}

// Helper to safely extract UUID from gin context
func getIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	// Handle if it's already a UUID object
	if id, ok := val.(uuid.UUID); ok {
		return id, true
	}

	// Handle if it's a string
	if str, ok := val.(string); ok {
		id, err := uuid.Parse(str)
		if err == nil {
			return id, true
		}
	}

	return uuid.Nil, false
}

func (h *ClientHandler) CreateProfile(c *gin.Context) {
	var req dto.CreateClientProfileDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.Service.Create(c.Request.Context(), userID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "profile created"})
}

func (h *ClientHandler) GetMyProfile(c *gin.Context) {
	userID, ok := getIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	data, err := h.Service.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *ClientHandler) UpdateProfile(c *gin.Context) {
	var req dto.UpdateClientProfileDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.Service.Update(c.Request.Context(), userID, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h *ClientHandler) DeleteProfile(c *gin.Context) {
	userID, ok := getIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.Service.Delete(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile deleted, Note that..you may recover your profile within 60 days"})
}

func (h *ClientHandler) RecoverProfile(c *gin.Context) {
	userID, ok := getIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}

	if err := h.Service.Recover(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile recovered"})
}

func (h *ClientHandler) AdminGetOne(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id format"})
		return
	}

	data, err := h.Service.AdminGetOne(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *ClientHandler) AdminList(c *gin.Context) {
	var query dto.AdminListClientsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := h.Service.AdminList(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}
