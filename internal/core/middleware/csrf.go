package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Skip check for safe methods (GET, HEAD, OPTIONS)
		if c.Request.Method == "GET" || c.Request.Method == "OPTIONS" || c.Request.Method == "HEAD" {
			c.Next()
			return
		}

		// 2. Get token from the secure Cookie
		cookieToken, err := c.Cookie("csrf_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF cookie missing"})
			return
		}

		// 3. Get token from the Header (Sent by your frontend/mobile app)
		headerToken := c.GetHeader("X-CSRF-Token")

		// 4. Compare them
		if headerToken == "" || headerToken != cookieToken {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF token mismatch or missing"})
			return
		}

		c.Next()
	}
}
