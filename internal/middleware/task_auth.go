package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/database"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
	"github.com/yukikurage/task-management-api/internal/models"
)

// RequireTaskAccess checks if the user has access to a task
// User must be a member of the task's organization
func RequireTaskAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get task ID from URL parameter
		taskIDStr := c.Param("id")
		taskID, err := strconv.ParseUint(taskIDStr, 10, 64)
		if err != nil {
			apierrors.BadRequest(c, "Invalid task ID")
			c.Abort()
			return
		}

		// Get current user ID
		userID, exists := GetUserID(c)
		if !exists {
			apierrors.Unauthorized(c, "")
			c.Abort()
			return
		}

		// Check if task exists (minimal data for authorization check)
		var task models.Task
		if err := database.GetDB().First(&task, taskID).Error; err != nil {
			apierrors.NotFound(c, "Task not found")
			c.Abort()
			return
		}

		// Check if user is a member of the task's organization
		var member models.OrganizationMember
		err = database.GetDB().
			Where("organization_id = ? AND user_id = ?", task.OrganizationID, userID).
			First(&member).Error
		if err != nil {
			// Return 404 instead of 403 to avoid leaking task existence
			apierrors.NotFound(c, "Task not found")
			c.Abort()
			return
		}

		c.Set(constants.ContextKeyTask, task)
		c.Next()
	}
}
