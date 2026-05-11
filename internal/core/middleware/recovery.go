package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and the stack trace (where it happened)
				log.Printf("PANIC RECOVERED: %v\n%s", err, debug.Stack())

				// Respond with a clean JSON error instead of crashing
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "An unexpected internal error occurred. Please try again later.",
				})
			}
		}()
		c.Next()
	}
}
