package repository

import (
	"time"

	"github.com/yukikurage/task-management-api/internal/models"
)

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	// Create creates a new task
	Create(task *models.Task) error

	// FindByID finds a task by ID with optional preloading
	FindByID(id uint64, preload ...string) (*models.Task, error)

	// List retrieves tasks with filtering and pagination
	List(filter TaskFilter) ([]models.Task, int64, error)

	// Update updates a task
	Update(task *models.Task) error

	// Delete soft deletes a task
	Delete(id uint64) error

	// AssignUsers assigns multiple users to a task
	AssignUsers(taskID uint64, userIDs []uint64) error

	// UnassignUsers removes user assignments from a task
	UnassignUsers(taskID uint64, userIDs []uint64) error

	// FindAssignment finds a specific task assignment
	FindAssignment(taskID, userID uint64) (*models.TaskAssignment, error)

	// CountUsersByIDs counts how many of the given user IDs exist
	CountUsersByIDs(userIDs []uint64, organizationID uint64) (int64, error)
}

// TaskFilter holds filtering options for listing tasks
type TaskFilter struct {
	OrganizationIDs []uint64
	Status          *models.TaskStatus
	CreatorID       *uint64
	AssignedUserID  *uint64
	DueDateFrom     *time.Time
	DueDateTo       *time.Time
	SortByDueDate   bool
	Page            int
	PageSize        int
}

// OrganizationRepository defines the interface for organization data access
type OrganizationRepository interface {
	// Create creates a new organization
	Create(org *models.Organization) error

	// FindByID finds an organization by ID
	FindByID(id uint64) (*models.Organization, error)

	// FindByInviteCode finds an organization by invite code
	FindByInviteCode(code string) (*models.Organization, error)

	// Update updates an organization
	Update(org *models.Organization) error

	// Delete deletes an organization and all related data
	Delete(id uint64) error

	// AddMember adds a member to an organization
	AddMember(member *models.OrganizationMember) error

	// RemoveMember removes a member from an organization
	RemoveMember(organizationID, userID uint64) error

	// FindMember finds a specific organization member
	FindMember(organizationID, userID uint64) (*models.OrganizationMember, error)

	// ListMembersByUserID lists all organizations a user is a member of
	ListMembersByUserID(userID uint64) ([]models.OrganizationMember, error)

	// ListMembers lists all members of an organization
	ListMembers(organizationID uint64) ([]models.OrganizationMember, error)
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(user *models.User) error

	// CreateWithPersonalOrganization creates a user, their personal organization,
	// and corresponding membership within a single transaction.
	CreateWithPersonalOrganization(user *models.User, org *models.Organization, member *models.OrganizationMember) error

	// FindByID finds a user by ID
	FindByID(id uint64) (*models.User, error)

	// FindByUsername finds a user by username
	FindByUsername(username string) (*models.User, error)
}
