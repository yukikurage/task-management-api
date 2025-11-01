package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/database"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
	"github.com/yukikurage/task-management-api/internal/models"
)

// RequireOrganizationAccess checks if the user is a member of the organization
func RequireOrganizationAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get organization ID from URL parameter
		orgIDStr := c.Param("id")
		orgID, err := strconv.ParseUint(orgIDStr, 10, 64)
		if err != nil {
			apierrors.BadRequest(c, "Invalid organization ID")
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

		// Check if organization exists
		var org models.Organization
		if err := database.GetDB().First(&org, orgID).Error; err != nil {
			apierrors.NotFound(c, "Organization not found")
			c.Abort()
			return
		}

		// Check if user is a member
		var member models.OrganizationMember
		err = database.GetDB().Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error
		if err != nil {
			// Return 404 instead of 403 to avoid leaking organization existence
			apierrors.NotFound(c, "Organization not found")
			c.Abort()
			return
		}

		// Store organization and membership in context
		c.Set(constants.ContextKeyOrganization, org)
		c.Set(constants.ContextKeyOrganizationMember, member)
		c.Next()
	}
}

// RequireOrganizationOwner checks if the user is an owner of the organization
func RequireOrganizationOwner() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get organization member from context (set by RequireOrganizationAccess)
		memberInterface, exists := c.Get(constants.ContextKeyOrganizationMember)
		if !exists {
			apierrors.Forbidden(c, "Organization access required")
			c.Abort()
			return
		}

		member, ok := memberInterface.(models.OrganizationMember)
		if !ok {
			apierrors.InternalError(c, "Invalid organization member data")
			c.Abort()
			return
		}

		// Check if user is owner
		if member.Role != models.RoleOwner {
			apierrors.Forbidden(c, "Only organization owners can perform this action")
			c.Abort()
			return
		}

		c.Next()
	}
}
