package repository

import (
	"errors"
	"fmt"

	"github.com/yukikurage/task-management-api/internal/models"
	"gorm.io/gorm"
)

// GormUserRepository is a GORM implementation of UserRepository
type GormUserRepository struct {
	db *gorm.DB
}

var (
	// ErrCreateUser is returned when creating a user fails inside the signup transaction.
	ErrCreateUser = errors.New("user repository: create user failed")
	// ErrCreateOrganization is returned when creating an organization fails inside the signup transaction.
	ErrCreateOrganization = errors.New("user repository: create organization failed")
	// ErrCreateOrganizationMember is returned when creating an organization member fails inside the signup transaction.
	ErrCreateOrganizationMember = errors.New("user repository: create organization member failed")
)

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

// Create creates a new user
func (r *GormUserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// CreateWithPersonalOrganization creates a user, a personal organization, and the membership atomically.
func (r *GormUserRepository) CreateWithPersonalOrganization(user *models.User, org *models.Organization, member *models.OrganizationMember) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrCreateUser, err)
		}

		if err := tx.Create(org).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrCreateOrganization, err)
		}

		member.OrganizationID = org.ID
		member.UserID = user.ID

		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("%w: %v", ErrCreateOrganizationMember, err)
		}

		return nil
	})
}

// FindByID finds a user by ID
func (r *GormUserRepository) FindByID(id uint64) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username
func (r *GormUserRepository) FindByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
