package repository

import (
	"github.com/yukikurage/task-management-api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormTaskRepository is a GORM implementation of TaskRepository
type GormTaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository creates a new TaskRepository
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &GormTaskRepository{db: db}
}

// Create creates a new task
func (r *GormTaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// FindByID finds a task by ID with optional preloading
func (r *GormTaskRepository) FindByID(id uint64, preload ...string) (*models.Task, error) {
	var task models.Task
	query := r.db

	// Apply preloading if specified
	for _, p := range preload {
		query = query.Preload(p)
	}

	if err := query.First(&task, id).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// List retrieves tasks with filtering and pagination
func (r *GormTaskRepository) List(filter TaskFilter) ([]models.Task, int64, error) {
	var tasks []models.Task

	if len(filter.OrganizationIDs) == 0 {
		return []models.Task{}, 0, nil
	}

	query := r.db.Model(&models.Task{}).Where("tasks.organization_id IN ?", filter.OrganizationIDs)

	// Apply filters
	if filter.Status != nil {
		query = query.Where("tasks.status = ?", *filter.Status)
	}
	if filter.CreatorID != nil {
		query = query.Where("tasks.creator_id = ?", *filter.CreatorID)
	}
	if filter.AssignedUserID != nil {
		assignmentSubQuery := r.db.Model(&models.TaskAssignment{}).
			Select("1").
			Where("task_assignments.task_id = tasks.id").
			Where("task_assignments.user_id = ?", *filter.AssignedUserID).
			Where("task_assignments.deleted_at IS NULL")
		query = query.Where("EXISTS (?)", assignmentSubQuery)
	}
	if filter.DueDateFrom != nil {
		query = query.Where("tasks.due_date >= ?", *filter.DueDateFrom)
	}
	if filter.DueDateTo != nil {
		query = query.Where("tasks.due_date < ?", *filter.DueDateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	listQuery := query
	if filter.SortByDueDate {
		listQuery = listQuery.Order("CASE WHEN tasks.due_date IS NULL THEN 1 ELSE 0 END, tasks.due_date ASC")
	} else {
		listQuery = listQuery.Order("tasks.created_at DESC")
	}

	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		listQuery = listQuery.Offset(offset).Limit(filter.PageSize)
	}

	if err := listQuery.Preload("Creator").Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Update updates a task
func (r *GormTaskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

// Delete soft deletes a task
func (r *GormTaskRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", id).Delete(&models.TaskAssignment{}).Error; err != nil {
			return err
		}

		return tx.Delete(&models.Task{}, id).Error
	})
}

// AssignUsers assigns multiple users to a task
func (r *GormTaskRepository) AssignUsers(taskID uint64, userIDs []uint64) error {
	assignments := make([]models.TaskAssignment, len(userIDs))

	for i, userID := range userIDs {
		assignments[i] = models.TaskAssignment{
			TaskID: taskID,
			UserID: userID,
		}
	}

	return r.db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "task_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"deleted_at": gorm.Expr("NULL")}),
		}).
		Create(&assignments).Error
}

// UnassignUsers removes user assignments from a task
func (r *GormTaskRepository) UnassignUsers(taskID uint64, userIDs []uint64) error {
	return r.db.Where("task_id = ? AND user_id IN ?", taskID, userIDs).
		Delete(&models.TaskAssignment{}).Error
}

// FindAssignment finds a specific task assignment
func (r *GormTaskRepository) FindAssignment(taskID, userID uint64) (*models.TaskAssignment, error) {
	var assignment models.TaskAssignment
	if err := r.db.Where("task_id = ? AND user_id = ?", taskID, userID).
		First(&assignment).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

// CountUsersByIDs counts how many of the given user IDs exist in the organization
func (r *GormTaskRepository) CountUsersByIDs(userIDs []uint64, organizationID uint64) (int64, error) {
	var count int64

	// Count users that are members of the organization
	err := r.db.Model(&models.User{}).
		Joins("JOIN organization_members ON users.id = organization_members.user_id").
		Where("organization_members.organization_id = ? AND users.id IN ?", organizationID, userIDs).
		Count(&count).Error

	return count, err
}
