package middleware

import (
	"log"
	"net/http"
	"strings"

	"rest_api/internal/core/utils"
	"rest_api/internal/services" // Import services instead of repository

	"github.com/gin-gonic/gin"
)

// AuthMiddleware now accepts services.AuthService to match your routes.go
func AuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Get header
		authHeader := c.GetHeader("Authorization")
		log.Printf("DEBUG: Authorization Header: [%s]", authHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Please login to continue"})
			return
		}

		// 2. Extract token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid login format"})
			return
		}
		tokenString := parts[1]

		// 3. Blacklist check
		// We use the authService to check if the token is valid/blacklisted
		isBlacklisted, err := authService.IsTokenBlacklisted(c.Request.Context(), tokenString)
		if err != nil || isBlacklisted {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token revoked. Please login again."})
			return
		}

		// 4. Validate claims
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		// 5. Database check (This is where Line 49 was panicking)
		// We fetch the user from the DB to ensure they still exist and are active
		user, err := authService.GetMe(c.Request.Context(), claims.UserID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User account no longer active"})
			return
		}

		// --- Consistent Naming for Context Keys ---
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("user", user) // Store the full user object for easy access in handlers

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied. Admin privileges required."})
			return
		}
		c.Next()
	}
}

func NannyOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "nanny" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Nanny accounts only."})
			return
		}
		c.Next()
	}
}

func EmployerOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "employer" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Client accounts only."})
			return
		}
		c.Next()
	}
}
