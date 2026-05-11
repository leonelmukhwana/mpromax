package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Example usage:
// r.GET("/admin/stats", RoleMiddleware("ADMIN"))
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {

		userRoleRaw, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "role not found in context",
			})
			return
		}

		// 🔐 safe type assertion
		userRole, ok := userRoleRaw.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "invalid role format",
			})
			return
		}

		// normalize role
		userRole = strings.ToUpper(strings.TrimSpace(userRole))

		// check allowed roles
		for _, r := range allowedRoles {
			if userRole == strings.ToUpper(r) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "forbidden: insufficient permissions",
		})
	}
}
