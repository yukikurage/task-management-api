package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
)

// RequireAuth checks if the user is authenticated via session
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get(constants.ContextKeyUserID)

		if userID == nil {
			apierrors.Unauthorized(c, "")
			c.Abort()
			return
		}

		// Store user ID in context for easy access in handlers
		c.Set(constants.ContextKeyUserID, userID)
		c.Next()
	}
}

// GetUserID retrieves the current user ID from context
func GetUserID(c *gin.Context) (uint64, bool) {
	userID, exists := c.Get(constants.ContextKeyUserID)
	if !exists {
		return 0, false
	}

	switch v := userID.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint64(v), true
	default:
		return 0, false
	}
}
