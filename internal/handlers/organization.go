package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/dto"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/services"
)

// OrganizationHandler handles HTTP requests for organizations.
type OrganizationHandler struct {
	orgService *services.OrganizationService
}

// NewOrganizationHandler creates a new OrganizationHandler.
func NewOrganizationHandler(orgService *services.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{
		orgService: orgService,
	}
}

// CreateOrganization creates a new organization for the authenticated user.
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	type CreateOrgRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var req CreateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	org, err := h.orgService.CreateOrganization(services.CreateOrganizationInput{
		Name:    req.Name,
		OwnerID: userID,
	})
	if err != nil {
		respondOrganizationError(c, err, "Failed to create organization")
		return
	}

	orgDTO := dto.ToOrganizationDTO(*org, true)
	c.JSON(http.StatusCreated, orgDTO)
}

// ListOrganizations returns all organizations the user belongs to.
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	memberships, err := h.orgService.ListOrganizationsForUser(userID)
	if err != nil {
		respondOrganizationError(c, err, "Failed to fetch organizations")
		return
	}

	orgs := make([]dto.OrganizationWithRoleDTO, len(memberships))
	for i, membership := range memberships {
		orgs[i] = dto.ToOrganizationWithRoleDTO(membership)
	}

	c.JSON(http.StatusOK, gin.H{
		"organizations": orgs,
	})
}

// GetOrganization returns organization details including members.
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	org, ok := getOrganizationFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization not found in context")
		return
	}

	member, ok := getOrganizationMemberFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization member not found in context")
		return
	}

	orgModel, members, err := h.orgService.GetOrganizationWithMembers(org.ID)
	if err != nil {
		respondOrganizationError(c, err, "Failed to fetch organization")
		return
	}

	detail := dto.ToOrganizationDetailDTO(*orgModel, members, member.Role)
	c.JSON(http.StatusOK, detail)
}

// UpdateOrganization updates organization attributes (currently name).
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	org, ok := getOrganizationFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization not found in context")
		return
	}

	type UpdateOrgRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var req UpdateOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	updatedOrg, err := h.orgService.UpdateOrganizationName(org.ID, req.Name)
	if err != nil {
		respondOrganizationError(c, err, "Failed to update organization")
		return
	}

	orgDTO := dto.ToOrganizationDTO(*updatedOrg, true)
	c.JSON(http.StatusOK, orgDTO)
}

// DeleteOrganization deletes an organization.
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	org, ok := getOrganizationFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization not found in context")
		return
	}

	if err := h.orgService.DeleteOrganization(org.ID); err != nil {
		respondOrganizationError(c, err, "Failed to delete organization")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Organization deleted successfully",
	})
}

// JoinOrganization allows a user to join via invite code.
func (h *OrganizationHandler) JoinOrganization(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	type JoinRequest struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}

	var req JoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	org, err := h.orgService.JoinOrganizationByInvite(userID, req.InviteCode)
	if err != nil {
		respondOrganizationError(c, err, "Failed to join organization")
		return
	}

	orgDTO := dto.ToOrganizationDTO(*org, true)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Successfully joined organization",
		"organization": orgDTO,
	})
}

// RegenerateInviteCode generates a new invite code for the organization.
func (h *OrganizationHandler) RegenerateInviteCode(c *gin.Context) {
	org, ok := getOrganizationFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization not found in context")
		return
	}

	updatedOrg, err := h.orgService.RegenerateInviteCode(org.ID)
	if err != nil {
		respondOrganizationError(c, err, "Failed to regenerate invite code")
		return
	}

	orgDTO := dto.ToOrganizationDTO(*updatedOrg, true)
	c.JSON(http.StatusOK, orgDTO)
}

// RemoveMember removes a member from the organization.
func (h *OrganizationHandler) RemoveMember(c *gin.Context) {
	org, ok := getOrganizationFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Organization not found in context")
		return
	}

	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	targetUserIDParam := c.Param("user_id")
	targetID, err := strconv.ParseUint(targetUserIDParam, 10, 64)
	if err != nil {
		apierrors.BadRequest(c, "Invalid user ID")
		return
	}

	if err := h.orgService.RemoveMember(org.ID, userID, targetID); err != nil {
		respondOrganizationError(c, err, "Failed to remove member")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}

func getOrganizationFromContext(c *gin.Context) (models.Organization, bool) {
	orgInterface, exists := c.Get(constants.ContextKeyOrganization)
	if !exists {
		return models.Organization{}, false
	}

	org, ok := orgInterface.(models.Organization)
	return org, ok
}

func getOrganizationMemberFromContext(c *gin.Context) (models.OrganizationMember, bool) {
	memberInterface, exists := c.Get(constants.ContextKeyOrganizationMember)
	if !exists {
		return models.OrganizationMember{}, false
	}

	member, ok := memberInterface.(models.OrganizationMember)
	return member, ok
}

func respondOrganizationError(c *gin.Context, err error, defaultMessage string) {
	switch {
	case err == nil:
		return
	case errors.Is(err, services.ErrInvalidOrganizationName),
		errors.Is(err, services.ErrCannotRemoveYourself):
		apierrors.BadRequest(c, err.Error())
	case errors.Is(err, services.ErrAlreadyOrganizationMember):
		apierrors.Conflict(c, err.Error())
	case errors.Is(err, services.ErrOrganizationNotFound),
		errors.Is(err, services.ErrOrganizationMemberNotFound),
		errors.Is(err, services.ErrInvalidInviteCode):
		apierrors.NotFound(c, err.Error())
	case errors.Is(err, services.ErrInviteCodeGenerationFailed):
		apierrors.InternalError(c, err.Error())
	default:
		apierrors.InternalError(c, defaultMessage)
	}
}
