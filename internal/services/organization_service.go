package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/repository"
	"github.com/yukikurage/task-management-api/internal/utils"
	"gorm.io/gorm"
)

var (
	ErrOrganizationNotFound       = errors.New("organization not found")
	ErrInvalidOrganizationName    = errors.New("organization name cannot be empty")
	ErrInviteCodeGenerationFailed = errors.New("failed to generate invite code")
	ErrInvalidInviteCode          = errors.New("invalid invite code")
	ErrAlreadyOrganizationMember  = errors.New("user is already a member of this organization")
	ErrCannotRemoveYourself       = errors.New("cannot remove yourself from the organization")
	ErrOrganizationMemberNotFound = errors.New("organization member not found")
)

// OrganizationService provides business logic for organization operations.
type OrganizationService struct {
	orgRepo repository.OrganizationRepository
}

// NewOrganizationService creates a new OrganizationService.
func NewOrganizationService(orgRepo repository.OrganizationRepository) *OrganizationService {
	return &OrganizationService{
		orgRepo: orgRepo,
	}
}

// CreateOrganizationInput represents parameters to create a new organization.
type CreateOrganizationInput struct {
	Name    string
	OwnerID uint64
}

// CreateOrganization creates a new organization and assigns the owner.
func (s *OrganizationService) CreateOrganization(input CreateOrganizationInput) (*models.Organization, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, ErrInvalidOrganizationName
	}

	inviteCode, err := utils.GenerateInviteCode()
	if err != nil {
		return nil, ErrInviteCodeGenerationFailed
	}

	org := &models.Organization{
		Name:       input.Name,
		InviteCode: inviteCode,
	}

	if err := s.orgRepo.Create(org); err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         input.OwnerID,
		Role:           models.RoleOwner,
		JoinedAt:       time.Now(),
	}

	if err := s.orgRepo.AddMember(member); err != nil {
		return nil, fmt.Errorf("failed to add owner to organization: %w", err)
	}

	return org, nil
}

// ListOrganizationsForUser returns organizations the user belongs to.
func (s *OrganizationService) ListOrganizationsForUser(userID uint64) ([]models.OrganizationMember, error) {
	memberships, err := s.orgRepo.ListMembersByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return memberships, nil
}

// GetOrganizationWithMembers returns an organization and all of its members.
func (s *OrganizationService) GetOrganizationWithMembers(orgID uint64) (*models.Organization, []models.OrganizationMember, error) {
	org, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrOrganizationNotFound
		}
		return nil, nil, fmt.Errorf("failed to find organization: %w", err)
	}

	members, err := s.orgRepo.ListMembers(orgID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list organization members: %w", err)
	}

	return org, members, nil
}

// UpdateOrganizationName updates an organization's name.
func (s *OrganizationService) UpdateOrganizationName(orgID uint64, name string) (*models.Organization, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidOrganizationName
	}

	org, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	org.Name = name
	if err := s.orgRepo.Update(org); err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return org, nil
}

// DeleteOrganization removes an organization.
func (s *OrganizationService) DeleteOrganization(orgID uint64) error {
	// Ensure organization exists
	if _, err := s.orgRepo.FindByID(orgID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("failed to find organization: %w", err)
	}

	if err := s.orgRepo.Delete(orgID); err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}

// JoinOrganizationByInvite adds a user to an organization via invite code.
func (s *OrganizationService) JoinOrganizationByInvite(userID uint64, inviteCode string) (*models.Organization, error) {
	org, err := s.orgRepo.FindByInviteCode(inviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidInviteCode
		}
		return nil, fmt.Errorf("failed to find organization by invite code: %w", err)
	}

	if _, err := s.orgRepo.FindMember(org.ID, userID); err == nil {
		return nil, ErrAlreadyOrganizationMember
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to verify membership: %w", err)
	}

	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleMember,
		JoinedAt:       time.Now(),
	}

	if err := s.orgRepo.AddMember(member); err != nil {
		return nil, fmt.Errorf("failed to add member to organization: %w", err)
	}

	return org, nil
}

// RegenerateInviteCode generates a new invite code for the organization.
func (s *OrganizationService) RegenerateInviteCode(orgID uint64) (*models.Organization, error) {
	org, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to find organization: %w", err)
	}

	code, err := utils.GenerateInviteCode()
	if err != nil {
		return nil, ErrInviteCodeGenerationFailed
	}

	org.InviteCode = code
	if err := s.orgRepo.Update(org); err != nil {
		return nil, fmt.Errorf("failed to update invite code: %w", err)
	}

	return org, nil
}

// RemoveMember removes a member from the organization.
func (s *OrganizationService) RemoveMember(orgID, actorID, targetID uint64) error {
	if targetID == actorID {
		return ErrCannotRemoveYourself
	}

	if _, err := s.orgRepo.FindMember(orgID, targetID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOrganizationMemberNotFound
		}
		return fmt.Errorf("failed to find organization member: %w", err)
	}

	if err := s.orgRepo.RemoveMember(orgID, targetID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	return nil
}
