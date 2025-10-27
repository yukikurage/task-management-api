package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/utils"
)

type OrganizationHandler struct{}

func NewOrganizationHandler() *OrganizationHandler {
	return &OrganizationHandler{}
}

// CreateOrganization creates a new organization
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type CreateOrgRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate invite code
	inviteCode, err := utils.GenerateInviteCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite code"})
		return
	}

	// Create organization
	org := models.Organization{
		Name:       req.Name,
		InviteCode: inviteCode,
	}

	if err := database.GetDB().Create(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	// Add creator as owner
	member := models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleOwner,
		JoinedAt:       time.Now(),
	}

	if err := database.GetDB().Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to organization"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

// ListOrganizations returns all organizations the user is a member of
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var memberships []models.OrganizationMember
	if err := database.GetDB().
		Preload("Organization").
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organizations"})
		return
	}

	// Extract organizations with role
	type OrgWithRole struct {
		models.Organization
		Role models.OrganizationRole `json:"role"`
	}

	orgsWithRole := make([]OrgWithRole, len(memberships))
	for i, m := range memberships {
		orgsWithRole[i] = OrgWithRole{
			Organization: m.Organization,
			Role:         m.Role,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"organizations": orgsWithRole,
	})
}

// GetOrganization returns organization details
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	// Organization is already loaded by RequireOrganizationAccess middleware
	orgInterface, _ := c.Get("organization")
	org := orgInterface.(models.Organization)

	memberInterface, _ := c.Get("organization_member")
	member := memberInterface.(models.OrganizationMember)

	// Load members
	var members []models.OrganizationMember
	database.GetDB().
		Preload("User").
		Where("organization_id = ?", org.ID).
		Find(&members)

	c.JSON(http.StatusOK, gin.H{
		"organization": org,
		"members":      members,
		"your_role":    member.Role,
	})
}

// UpdateOrganization updates organization name
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	orgInterface, exists := c.Get("organization")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization not found in context"})
		return
	}

	org, ok := orgInterface.(models.Organization)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organization"})
		return
	}

	type UpdateOrgRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	org.Name = req.Name
	if err := database.GetDB().Save(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// DeleteOrganization deletes an organization
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	orgInterface, _ := c.Get("organization")
	org := orgInterface.(models.Organization)

	// Delete all tasks in the organization
	if err := database.GetDB().Where("organization_id = ?", org.ID).Delete(&models.Task{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization tasks"})
		return
	}

	// Delete all members
	if err := database.GetDB().Where("organization_id = ?", org.ID).Delete(&models.OrganizationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization members"})
		return
	}

	// Delete organization
	if err := database.GetDB().Delete(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Organization deleted successfully",
	})
}

// JoinOrganization allows a user to join via invite code
func (h *OrganizationHandler) JoinOrganization(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type JoinRequest struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}

	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find organization by invite code
	var org models.Organization
	if err := database.GetDB().Where("invite_code = ?", req.InviteCode).First(&org).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid invite code"})
		return
	}

	// Check if already a member
	var existing models.OrganizationMember
	err := database.GetDB().Where("organization_id = ? AND user_id = ?", org.ID, userID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Already a member of this organization"})
		return
	}

	// Add as member
	member := models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleMember,
		JoinedAt:       time.Now(),
	}

	if err := database.GetDB().Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join organization"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Successfully joined organization",
		"organization": org,
	})
}

// RegenerateInviteCode generates a new invite code for the organization
func (h *OrganizationHandler) RegenerateInviteCode(c *gin.Context) {
	orgInterface, _ := c.Get("organization")
	org := orgInterface.(models.Organization)

	// Generate new invite code
	inviteCode, err := utils.GenerateInviteCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invite code"})
		return
	}

	org.InviteCode = inviteCode
	if err := database.GetDB().Save(&org).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update invite code"})
		return
	}

	c.JSON(http.StatusOK, org)
}

// RemoveMember removes a member from the organization
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	orgInterface, _ := c.Get("organization")
	org := orgInterface.(models.Organization)

	targetUserID := c.Param("user_id")

	// Cannot remove yourself
	currentUserID, _ := middleware.GetUserID(c)
	if targetUserID == string(rune(currentUserID)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot remove yourself"})
		return
	}

	// Delete member
	if err := database.GetDB().
		Where("organization_id = ? AND user_id = ?", org.ID, targetUserID).
		Delete(&models.OrganizationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}
