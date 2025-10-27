package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	UserIDKey = "user_id"
)

// RequireAuth checks if the user is authenticated via session
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(UserIDKey)

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		// Store user ID in context for easy access in handlers
		c.Set(UserIDKey, userID)
		c.Next()
	}
}

// GetUserID retrieves the current user ID from context
func GetUserID(c *gin.Context) (uint64, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, false
	}

	uid, ok := userID.(uint64)
	return uid, ok
}
