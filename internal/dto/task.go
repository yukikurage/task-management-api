package dto

import (
	"time"

	"github.com/yukikurage/task-management-api/internal/models"
)

// UserDTO represents a user in API responses
type UserDTO struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// OrganizationDTO represents an organization in API responses
type OrganizationDTO struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	InviteCode string `json:"invite_code,omitempty"`
}

// TaskAssignmentDTO represents a task assignment in API responses
type TaskAssignmentDTO struct {
	User UserDTO `json:"user"`
}

// TaskDTO represents a task in API responses
type TaskDTO struct {
	ID             uint64              `json:"id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Status         models.TaskStatus   `json:"status"`
	DueDate        *time.Time          `json:"due_date"`
	CreatorID      uint64              `json:"creator_id"`
	OrganizationID uint64              `json:"organization_id"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Creator        *UserDTO            `json:"creator,omitempty"`
	Organization   *OrganizationDTO    `json:"organization,omitempty"`
	Assignments    []TaskAssignmentDTO `json:"assignments,omitempty"`
}

// TaskListItemDTO represents a task in list responses (minimal data)
type TaskListItemDTO struct {
	ID          uint64            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      models.TaskStatus `json:"status"`
	DueDate     *time.Time        `json:"due_date"`
	CreatorID   uint64            `json:"creator_id"`
	Creator     *UserDTO          `json:"creator,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// TaskListResponse represents a paginated list of tasks
type TaskListResponse struct {
	Tasks      []TaskListItemDTO `json:"tasks"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalCount int64             `json:"total_count"`
	TotalPages int               `json:"total_pages"`
}

// Conversion functions

// ToUserDTO converts a User model to UserDTO
func ToUserDTO(user models.User) UserDTO {
	return UserDTO{
		ID:       user.ID,
		Username: user.Username,
	}
}

// ToOrganizationDTO converts an Organization model to OrganizationDTO
func ToOrganizationDTO(org models.Organization, includeInviteCode bool) OrganizationDTO {
	dto := OrganizationDTO{
		ID:   org.ID,
		Name: org.Name,
	}
	if includeInviteCode {
		dto.InviteCode = org.InviteCode
	}
	return dto
}

// ToTaskDTO converts a Task model to TaskDTO
func ToTaskDTO(task models.Task) TaskDTO {
	dto := TaskDTO{
		ID:             task.ID,
		Title:          task.Title,
		Description:    task.Description,
		Status:         task.Status,
		DueDate:        task.DueDate,
		CreatorID:      task.CreatorID,
		OrganizationID: task.OrganizationID,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}

	// Include creator if preloaded
	if task.Creator.ID != 0 {
		creator := ToUserDTO(task.Creator)
		dto.Creator = &creator
	}

	// Include organization if preloaded
	if task.Organization.ID != 0 {
		org := ToOrganizationDTO(task.Organization, false)
		dto.Organization = &org
	}

	// Include assignments if preloaded
	if len(task.Assignments) > 0 {
		dto.Assignments = make([]TaskAssignmentDTO, len(task.Assignments))
		for i, assignment := range task.Assignments {
			dto.Assignments[i] = TaskAssignmentDTO{
				User: ToUserDTO(assignment.User),
			}
		}
	}

	return dto
}

// ToTaskListItemDTO converts a Task model to TaskListItemDTO
func ToTaskListItemDTO(task models.Task) TaskListItemDTO {
	dto := TaskListItemDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		DueDate:     task.DueDate,
		CreatorID:   task.CreatorID,
		CreatedAt:   task.CreatedAt,
	}

	// Include creator if preloaded
	if task.Creator.ID != 0 {
		creator := ToUserDTO(task.Creator)
		dto.Creator = &creator
	}

	return dto
}

// ToTaskListResponse converts a slice of tasks to TaskListResponse
func ToTaskListResponse(tasks []models.Task, page, pageSize int, totalCount int64) TaskListResponse {
	items := make([]TaskListItemDTO, len(tasks))
	for i, task := range tasks {
		items[i] = ToTaskListItemDTO(task)
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	return TaskListResponse{
		Tasks:      items,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
