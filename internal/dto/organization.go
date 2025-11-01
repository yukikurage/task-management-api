package dto

import (
	"time"

	"github.com/yukikurage/task-management-api/internal/models"
)

// OrganizationWithRoleDTO represents an organization with the user's role
type OrganizationWithRoleDTO struct {
	OrganizationDTO
	Role models.OrganizationRole `json:"role"`
}

// OrganizationMemberDTO represents a member in an organization
type OrganizationMemberDTO struct {
	User     UserDTO                 `json:"user"`
	Role     models.OrganizationRole `json:"role"`
	JoinedAt time.Time               `json:"joined_at"`
}

// OrganizationDetailDTO represents detailed organization information
type OrganizationDetailDTO struct {
	OrganizationDTO
	Members  []OrganizationMemberDTO `json:"members"`
	YourRole models.OrganizationRole `json:"your_role"`
}

// ToOrganizationWithRoleDTO converts an organization member to DTO with role
func ToOrganizationWithRoleDTO(member models.OrganizationMember) OrganizationWithRoleDTO {
	return OrganizationWithRoleDTO{
		OrganizationDTO: ToOrganizationDTO(member.Organization, false),
		Role:            member.Role,
	}
}

// ToOrganizationMemberDTO converts a member to DTO
func ToOrganizationMemberDTO(member models.OrganizationMember) OrganizationMemberDTO {
	return OrganizationMemberDTO{
		User:     ToUserDTO(member.User),
		Role:     member.Role,
		JoinedAt: member.JoinedAt,
	}
}

// ToOrganizationDetailDTO converts organization with members to detailed DTO
func ToOrganizationDetailDTO(org models.Organization, members []models.OrganizationMember, yourRole models.OrganizationRole) OrganizationDetailDTO {
	memberDTOs := make([]OrganizationMemberDTO, len(members))
	for i, member := range members {
		memberDTOs[i] = ToOrganizationMemberDTO(member)
	}

	return OrganizationDetailDTO{
		OrganizationDTO: ToOrganizationDTO(org, true),
		Members:         memberDTOs,
		YourRole:        yourRole,
	}
}
