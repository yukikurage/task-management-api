package repository

import (
	"github.com/yukikurage/task-management-api/internal/models"
	"gorm.io/gorm"
)

// GormOrganizationRepository is a GORM implementation of OrganizationRepository
type GormOrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new OrganizationRepository
func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &GormOrganizationRepository{db: db}
}

// Create creates a new organization
func (r *GormOrganizationRepository) Create(org *models.Organization) error {
	return r.db.Create(org).Error
}

// FindByID finds an organization by ID
func (r *GormOrganizationRepository) FindByID(id uint64) (*models.Organization, error) {
	var org models.Organization
	if err := r.db.First(&org, id).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

// FindByInviteCode finds an organization by invite code
func (r *GormOrganizationRepository) FindByInviteCode(code string) (*models.Organization, error) {
	var org models.Organization
	if err := r.db.Where("invite_code = ?", code).First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

// Update updates an organization
func (r *GormOrganizationRepository) Update(org *models.Organization) error {
	return r.db.Save(org).Error
}

// Delete deletes an organization and all related data in a transaction
func (r *GormOrganizationRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete all tasks in the organization
		if err := tx.Where("organization_id = ?", id).Delete(&models.Task{}).Error; err != nil {
			return err
		}

		// Delete all members
		if err := tx.Where("organization_id = ?", id).Delete(&models.OrganizationMember{}).Error; err != nil {
			return err
		}

		// Delete organization
		if err := tx.Delete(&models.Organization{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

// AddMember adds a member to an organization
func (r *GormOrganizationRepository) AddMember(member *models.OrganizationMember) error {
	return r.db.Create(member).Error
}

// RemoveMember removes a member from an organization
func (r *GormOrganizationRepository) RemoveMember(organizationID, userID uint64) error {
	return r.db.Where("organization_id = ? AND user_id = ?", organizationID, userID).
		Delete(&models.OrganizationMember{}).Error
}

// FindMember finds a specific organization member
func (r *GormOrganizationRepository) FindMember(organizationID, userID uint64) (*models.OrganizationMember, error) {
	var member models.OrganizationMember
	if err := r.db.Where("organization_id = ? AND user_id = ?", organizationID, userID).
		First(&member).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

// ListMembersByUserID lists all organizations a user is a member of
func (r *GormOrganizationRepository) ListMembersByUserID(userID uint64) ([]models.OrganizationMember, error) {
	var memberships []models.OrganizationMember
	if err := r.db.Preload("Organization").
		Where("user_id = ?", userID).
		Find(&memberships).Error; err != nil {
		return nil, err
	}
	return memberships, nil
}

// ListMembers lists all members of an organization
func (r *GormOrganizationRepository) ListMembers(organizationID uint64) ([]models.OrganizationMember, error) {
	var members []models.OrganizationMember
	if err := r.db.Preload("User").
		Where("organization_id = ?", organizationID).
		Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}
