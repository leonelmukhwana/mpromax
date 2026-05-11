package handlers

import (
	"fmt"
	"net/http"
	"rest_api/internal/dto"
	"rest_api/internal/services"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(s services.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Check email/password requirements"})
		return
	}

	id, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": id,
		"message": "Account created. Check terminal for verification code.",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password required"})
		return
	}

	user, at, rt, needs2FA, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if needs2FA {
		c.JSON(http.StatusOK, gin.H{
			"needs_2fa": true,
			"message":   "Admin 2FA code sent to terminal",
		})
		return
	}

	// Set Refresh Token in HttpOnly Cookie
	c.SetCookie("refresh_token", rt, 604800, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token": at,
		"user":         user,
	})
}

// 3. VERIFY OTP - Critical for Email Verification and Admin 2FA
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Valid email, code, and type required"})
		return
	}

	at, rt, err := h.service.VerifyOTP(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.setTokenCookies(c, rt)
	c.JSON(http.StatusOK, gin.H{"message": "Verified successfully", "access_token": at})
}

// 4. RESEND OTP - With Rate Limiting (handled in service/repo)
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		Type  string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and OTP type required"})
		return
	}

	if err := h.service.ResendOTP(c.Request.Context(), req.Email, req.Type); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "A new code has been sent."})
}

// 5. FORGOT PASSWORD - Triggers a reset OTP
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email required"})
		return
	}

	// We return 200 even if email doesn't exist to prevent "User Enumeration" attacks
	_ = h.service.ForgotPassword(c.Request.Context(), req.Email)

	c.JSON(http.StatusOK, gin.H{"message": "If that email exists, a reset code has been sent."})
}

// 6. RESET PASSWORD - The final step using the OTP
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully. You can now login."})
}

// Helper to set secure cookies
func (h *AuthHandler) setTokenCookies(c *gin.Context, refreshToken string) {
	// HttpOnly: true (JS can't touch it), Secure: true (HTTPS only), SameSite: Strict (No CSRF)
	c.SetCookie("refresh_token", refreshToken, 604800, "/", "", true, true)
}

// logout handler
func (h *AuthHandler) Logout(c *gin.Context) {
	// 1. Extract the token from the Header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header is required"})
		return
	}

	// Split "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token format"})
		return
	}
	tokenString := parts[1]

	// 2. Call the service
	err := h.service.Logout(c.Request.Context(), tokenString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log out. Please try again."})
		return
	}

	// 3. Success
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully. Token revoked."})
}

// Block or unblock any user
func (h *AuthHandler) ToggleBlock(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var input struct {
		Block bool `json:"block"` // true to block, false to unblock
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.UpdateUserStatus(c.Request.Context(), userID, input.Block); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user status"})
		return
	}

	message := "User unblocked successfully"
	if input.Block {
		message = "User blocked successfully"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// list of registered users
func (h *AuthHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	users, total, totalPages, err := h.service.GetPaginatedUsers(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{
			"total_records": total,
			"total_pages":   totalPages,
			"current_page":  page,
			"limit":         limit,
		},
	})
}

// update email
// UpdateEmail handles the request for a user to change their own email address
func (h *AuthHandler) UpdateEmail(c *gin.Context) {
	// 1. Get the user_id from the context (set by your JWT/Auth middleware)
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Type assertion to convert the context value to a UUID
	userID, ok := val.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user identification"})
		return
	}

	// 2. Define the input structure for the request body
	var input struct {
		NewEmail string `json:"new_email" binding:"required,email"`
	}

	// 3. Bind the JSON body to the input struct
	if err := c.ShouldBindJSON(&input); err != nil {
		// ADD THIS LINE TO DEBUG:
		fmt.Printf("Received Input: %+v | Error: %v\n", input, err)

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}

	// 4. Call the service method we just registered in the interface
	err := h.service.UpdateUserEmail(c.Request.Context(), userID, input.NewEmail)
	if err != nil {
		// If the error is "email already in use", we return a 400 Bad Request
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Return success
	c.JSON(http.StatusOK, gin.H{
		"message": "Email updated successfully",
		"email":   input.NewEmail,
	})
}
