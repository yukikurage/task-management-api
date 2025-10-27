package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/database"
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
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid task ID",
			})
			c.Abort()
			return
		}

		// Get current user ID
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		// Check if task exists and load relations
		var task models.Task
		if err := database.GetDB().
			Preload("Creator").
			Preload("Organization").
			Preload("Assignments").
			Preload("Assignments.User").
			First(&task, taskID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
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
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			c.Abort()
			return
		}

		c.Set("task", task)
		c.Next()
	}
}
