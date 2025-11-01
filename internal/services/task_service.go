package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrNotOrganizationMember  = errors.New("user is not a member of the organization")
	ErrTaskNotFound           = errors.New("task not found")
	ErrNotTaskCreator         = errors.New("only the task creator can perform this action")
	ErrTaskPermissionDenied   = errors.New("user does not have permission to modify this task")
	ErrNoUserIDsProvided      = errors.New("at least one user ID is required")
	ErrTitleRequired          = errors.New("title is required")
	ErrTitleEmpty             = errors.New("title cannot be empty")
	ErrInvalidTaskAssignee    = errors.New("one or more users do not exist or are not members of the organization")
	ErrAIServiceNotConfigured = errors.New("AI service is not configured")
	ErrAINoTasksGenerated     = errors.New("AI did not generate any tasks")
	ErrAINoValidTasks         = errors.New("no valid tasks could be created from AI output")
)

// TaskService handles task business logic
type TaskService struct {
	taskRepo  repository.TaskRepository
	orgRepo   repository.OrganizationRepository
	aiService *AIService
}

// NewTaskService creates a new TaskService
func NewTaskService(taskRepo repository.TaskRepository, orgRepo repository.OrganizationRepository, aiService *AIService) *TaskService {
	return &TaskService{
		taskRepo:  taskRepo,
		orgRepo:   orgRepo,
		aiService: aiService,
	}
}

// ListTasksInput represents filters for listing tasks
type ListTasksInput struct {
	UserID         uint64
	OrganizationID *uint64
	AssignedToMe   bool
	DueToday       bool
	Status         *models.TaskStatus
	SortByDueDate  bool
	Page           int
	PageSize       int
}

// CreateTaskInput represents input for creating a task
type CreateTaskInput struct {
	Title          string
	Description    string
	Status         models.TaskStatus
	DueDate        *time.Time
	OrganizationID uint64
	CreatorID      uint64
}

// UpdateTaskInput represents input for updating a task
type UpdateTaskInput struct {
	Title        *string
	Description  *string
	Status       *models.TaskStatus
	DueDate      *time.Time
	ClearDueDate bool
}

// AssignUsersInput represents input for assigning users to a task
type AssignUsersInput struct {
	TaskID  uint64
	ActorID uint64
	UserIDs []uint64
}

// ListTasks returns tasks accessible to a user based on the provided filters
func (s *TaskService) ListTasks(input ListTasksInput) ([]models.Task, int64, error) {
	orgIDs, err := s.resolveAccessibleOrganizationIDs(input.UserID, input.OrganizationID)
	if err != nil {
		return nil, 0, err
	}

	if len(orgIDs) == 0 {
		return []models.Task{}, 0, nil
	}

	filter := repository.TaskFilter{
		OrganizationIDs: orgIDs,
		Page:            input.Page,
		PageSize:        input.PageSize,
		SortByDueDate:   input.SortByDueDate,
	}

	if input.Status != nil {
		filter.Status = input.Status
	}
	if input.AssignedToMe {
		filter.AssignedUserID = &input.UserID
	}
	if input.DueToday {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)
		filter.DueDateFrom = &startOfDay
		filter.DueDateTo = &endOfDay
	}

	tasks, total, err := s.taskRepo.List(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// GetTask returns a task with related data
func (s *TaskService) GetTask(taskID uint64) (*models.Task, error) {
	task, err := s.taskRepo.FindByID(taskID, "Creator", "Organization", "Assignments", "Assignments.User")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	return task, nil
}

// CreateTask creates a new task with validation and assigns the creator
func (s *TaskService) CreateTask(input CreateTaskInput) (*models.Task, error) {
	if input.Title == "" {
		return nil, ErrTitleRequired
	}

	if err := s.ensureOrganizationMember(input.OrganizationID, input.CreatorID); err != nil {
		return nil, err
	}

	if input.Status == "" {
		input.Status = models.TaskStatusTodo
	}

	task := &models.Task{
		Title:          input.Title,
		Description:    input.Description,
		Status:         input.Status,
		DueDate:        input.DueDate,
		OrganizationID: input.OrganizationID,
		CreatorID:      input.CreatorID,
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	if err := s.taskRepo.AssignUsers(task.ID, []uint64{input.CreatorID}); err != nil {
		return nil, fmt.Errorf("failed to assign creator to task: %w", err)
	}

	return s.taskRepo.FindByID(task.ID, "Creator", "Organization", "Assignments", "Assignments.User")
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(taskID uint64, input UpdateTaskInput) (*models.Task, error) {
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	if input.Title != nil {
		if *input.Title == "" {
			return nil, ErrTitleEmpty
		}
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Status != nil {
		task.Status = *input.Status
	}
	if input.ClearDueDate {
		task.DueDate = nil
	} else if input.DueDate != nil {
		task.DueDate = input.DueDate
	}

	if err := s.taskRepo.Update(task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return s.taskRepo.FindByID(task.ID, "Creator", "Organization", "Assignments", "Assignments.User")
}

// DeleteTask deletes a task if the actor is the creator
func (s *TaskService) DeleteTask(taskID, actorID uint64) error {
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to find task: %w", err)
	}

	if task.CreatorID != actorID {
		return ErrNotTaskCreator
	}

	if err := s.taskRepo.Delete(taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// AssignUsers assigns multiple users to a task with validation
func (s *TaskService) AssignUsers(input AssignUsersInput) error {
	if len(input.UserIDs) == 0 {
		return ErrNoUserIDsProvided
	}

	task, err := s.taskRepo.FindByID(input.TaskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to find task: %w", err)
	}

	if task.CreatorID != input.ActorID {
		return ErrNotTaskCreator
	}

	userIDs := uniqueUint64(input.UserIDs)

	count, err := s.taskRepo.CountUsersByIDs(userIDs, task.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to verify users: %w", err)
	}
	if int(count) != len(userIDs) {
		return ErrInvalidTaskAssignee
	}

	if err := s.taskRepo.AssignUsers(task.ID, userIDs); err != nil {
		return fmt.Errorf("failed to assign users: %w", err)
	}

	return nil
}

// UnassignUsers removes user assignments from a task
func (s *TaskService) UnassignUsers(taskID, actorID uint64, userIDs []uint64) error {
	if len(userIDs) == 0 {
		return ErrNoUserIDsProvided
	}

	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("failed to find task: %w", err)
	}

	if task.CreatorID != actorID {
		return ErrNotTaskCreator
	}

	uniqueIDs := uniqueUint64(userIDs)

	if err := s.taskRepo.UnassignUsers(taskID, uniqueIDs); err != nil {
		return fmt.Errorf("failed to unassign users: %w", err)
	}

	return nil
}

// ToggleTaskStatus toggles a task between todo and done
func (s *TaskService) ToggleTaskStatus(taskID, actorID uint64) (*models.Task, error) {
	task, err := s.taskRepo.FindByID(taskID, "Assignments")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	if task.CreatorID != actorID {
		// Ensure the actor is assigned to the task
		permitted := false
		for _, assignment := range task.Assignments {
			if assignment.UserID == actorID {
				permitted = true
				break
			}
		}
		if !permitted {
			return nil, ErrTaskPermissionDenied
		}
	}

	if task.Status == models.TaskStatusDone {
		task.Status = models.TaskStatusTodo
	} else {
		task.Status = models.TaskStatusDone
	}

	if err := s.taskRepo.Update(task); err != nil {
		return nil, fmt.Errorf("failed to toggle status: %w", err)
	}

	return task, nil
}

// GenerateTasksInput represents input for AI task generation
type GenerateTasksInput struct {
	Text      string
	CreatorID uint64
}

// GenerateTasks uses AI to generate tasks from text
func (s *TaskService) GenerateTasks(ctx context.Context, input GenerateTasksInput) ([]GeneratedTask, error) {
	if s.aiService == nil {
		return nil, ErrAIServiceNotConfigured
	}

	aiTasks, err := s.aiService.GenerateTasksFromText(ctx, input.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tasks: %w", err)
	}

	if len(aiTasks) == 0 {
		return nil, ErrAINoTasksGenerated
	}
	if len(aiTasks) > constants.MaxAIGeneratedTasks {
		return nil, fmt.Errorf("AI generated too many tasks (max %d)", constants.MaxAIGeneratedTasks)
	}

	validTasks := make([]GeneratedTask, 0, len(aiTasks))
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, aiTask := range aiTasks {
		if strings.TrimSpace(aiTask.Title) == "" {
			continue
		}

		if aiTask.DueDate != nil {
			if aiTask.DueDate.Before(cutoff) {
				aiTask.DueDate = nil
			}
		}

		validTasks = append(validTasks, aiTask)
	}

	if len(validTasks) == 0 {
		return nil, ErrAINoValidTasks
	}

	return validTasks, nil
}

// resolveAccessibleOrganizationIDs returns the organization IDs the user can access
func (s *TaskService) resolveAccessibleOrganizationIDs(userID uint64, organizationID *uint64) ([]uint64, error) {
	if organizationID != nil {
		if err := s.ensureOrganizationMember(*organizationID, userID); err != nil {
			return nil, err
		}
		return []uint64{*organizationID}, nil
	}

	memberships, err := s.orgRepo.ListMembersByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization memberships: %w", err)
	}

	orgIDs := make([]uint64, 0, len(memberships))
	for _, m := range memberships {
		orgIDs = append(orgIDs, m.OrganizationID)
	}

	return orgIDs, nil
}

// ensureOrganizationMember verifies that a user belongs to an organization
func (s *TaskService) ensureOrganizationMember(orgID, userID uint64) error {
	_, err := s.orgRepo.FindMember(orgID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotOrganizationMember
		}
		return fmt.Errorf("failed to verify organization membership: %w", err)
	}
	return nil
}

// uniqueUint64 removes duplicate values from a slice of uint64
func uniqueUint64(values []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(values))
	result := make([]uint64, 0, len(values))

	for _, v := range values {
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}

	return result
}
