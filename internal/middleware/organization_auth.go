package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/models"
)

// RequireOrganizationAccess checks if the user is a member of the organization
func RequireOrganizationAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get organization ID from URL parameter
		orgIDStr := c.Param("id")
		orgID, err := strconv.ParseUint(orgIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid organization ID",
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

		// Check if organization exists
		var org models.Organization
		if err := database.GetDB().First(&org, orgID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Organization not found",
			})
			c.Abort()
			return
		}

		// Check if user is a member
		var member models.OrganizationMember
		err = database.GetDB().Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error
		if err != nil {
			// Return 404 instead of 403 to avoid leaking organization existence
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Organization not found",
			})
			c.Abort()
			return
		}

		// Store organization and membership in context
		c.Set("organization", org)
		c.Set("organization_member", member)
		c.Next()
	}
}

// RequireOrganizationOwner checks if the user is an owner of the organization
func RequireOrganizationOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get organization member from context (set by RequireOrganizationAccess)
		memberInterface, exists := c.Get("organization_member")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Organization access required",
			})
			c.Abort()
			return
		}

		member, ok := memberInterface.(models.OrganizationMember)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid organization member data",
			})
			c.Abort()
			return
		}

		// Check if user is owner
		if member.Role != models.RoleOwner {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Only organization owners can perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
