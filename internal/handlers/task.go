package handlers

import (
	"context"
	stdErrors "errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/dto"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/services"
	"github.com/yukikurage/task-management-api/internal/utils"
)

// TaskHandler orchestrates task-related HTTP handlers.
type TaskHandler struct {
	taskService *services.TaskService
}

// NewTaskHandler creates a new TaskHandler.
func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

// ListTasks returns tasks accessible by the current user with optional filtering.
func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	var orgIDPtr *uint64
	if organizationIDStr := c.Query("organization_id"); organizationIDStr != "" {
		orgID, err := strconv.ParseUint(organizationIDStr, 10, 64)
		if err != nil {
			apierrors.BadRequest(c, "Invalid organization_id")
			return
		}
		orgIDPtr = &orgID
	}

	assignedToMe := c.Query("assigned_to_me") == "true"
	dueToday := c.Query("due_today") == "true"
	sortByDueDate := c.Query("sort") == "due_date"

	var statusPtr *models.TaskStatus
	if statusStr := c.Query("status"); statusStr != "" {
		status := models.TaskStatus(statusStr)
		if status != models.TaskStatusTodo && status != models.TaskStatusDone {
			apierrors.BadRequest(c, "Invalid status filter")
			return
		}
		statusPtr = &status
	}

	params := utils.GetPaginationParams(c)

	tasks, total, err := h.taskService.ListTasks(services.ListTasksInput{
		UserID:         userID,
		OrganizationID: orgIDPtr,
		AssignedToMe:   assignedToMe,
		DueToday:       dueToday,
		Status:         statusPtr,
		SortByDueDate:  sortByDueDate,
		Page:           params.Page,
		PageSize:       params.Limit,
	})
	if err != nil {
		switch {
		case stdErrors.Is(err, services.ErrNotOrganizationMember):
			apierrors.Forbidden(c, err.Error())
		default:
			apierrors.InternalError(c, "Failed to list tasks")
		}
		return
	}

	response := dto.ToTaskListResponse(tasks, params.Page, params.Limit, total)
	c.JSON(http.StatusOK, response)
}

// GetTask returns a task by ID.
func (h *TaskHandler) GetTask(c *gin.Context) {
	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	fullTask, err := h.taskService.GetTask(task.ID)
	if err != nil {
		respondTaskError(c, err, "Failed to fetch task")
		return
	}

	taskDTO := dto.ToTaskDTO(*fullTask)
	c.JSON(http.StatusOK, taskDTO)
}

// CreateTask creates a new task.
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	type CreateTaskRequest struct {
		Title          string     `json:"title" binding:"required"`
		Description    string     `json:"description"`
		Status         *string    `json:"status"`
		DueDate        *time.Time `json:"due_date"`
		OrganizationID uint64     `json:"organization_id" binding:"required"`
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	var status models.TaskStatus
	if req.Status != nil && *req.Status != "" {
		status = models.TaskStatus(*req.Status)
		if status != models.TaskStatusTodo && status != models.TaskStatusDone {
			apierrors.BadRequest(c, "Invalid status value")
			return
		}
	}

	task, err := h.taskService.CreateTask(services.CreateTaskInput{
		Title:          req.Title,
		Description:    req.Description,
		Status:         status,
		DueDate:        req.DueDate,
		OrganizationID: req.OrganizationID,
		CreatorID:      userID,
	})
	if err != nil {
		respondTaskError(c, err, "Failed to create task")
		return
	}

	taskDTO := dto.ToTaskDTO(*task)
	c.JSON(http.StatusCreated, taskDTO)
}

// UpdateTask updates an existing task.
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	if len(raw) == 0 {
		apierrors.BadRequest(c, "No fields to update")
		return
	}

	updateInput := services.UpdateTaskInput{}

	if titleVal, exists := raw["title"]; exists {
		title, ok := titleVal.(string)
		if !ok {
			apierrors.BadRequest(c, "Title must be a string")
			return
		}
		updateInput.Title = &title
	}

	if descVal, exists := raw["description"]; exists {
		description, ok := descVal.(string)
		if !ok {
			apierrors.BadRequest(c, "Description must be a string")
			return
		}
		updateInput.Description = &description
	}

	if statusVal, exists := raw["status"]; exists {
		statusStr, ok := statusVal.(string)
		if !ok {
			apierrors.BadRequest(c, "Status must be a string")
			return
		}
		status := models.TaskStatus(statusStr)
		if status != models.TaskStatusTodo && status != models.TaskStatusDone {
			apierrors.BadRequest(c, "Invalid status value")
			return
		}
		updateInput.Status = &status
	}

	if dueVal, exists := raw["due_date"]; exists {
		if dueVal == nil {
			updateInput.ClearDueDate = true
		} else if dueStr, ok := dueVal.(string); ok {
			parsed, err := time.Parse(time.RFC3339, dueStr)
			if err != nil {
				apierrors.BadRequest(c, "Invalid due_date format")
				return
			}
			updateInput.DueDate = &parsed
		} else {
			apierrors.BadRequest(c, "Invalid due_date value")
			return
		}
	}

	updatedTask, err := h.taskService.UpdateTask(task.ID, updateInput)
	if err != nil {
		respondTaskError(c, err, "Failed to update task")
		return
	}

	taskDTO := dto.ToTaskDTO(*updatedTask)
	c.JSON(http.StatusOK, taskDTO)
}

// DeleteTask deletes a task.
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	if err := h.taskService.DeleteTask(task.ID, userID); err != nil {
		respondTaskError(c, err, "Failed to delete task")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
	})
}

// AssignTask assigns users to a task.
func (h *TaskHandler) AssignTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	type AssignUsersRequest struct {
		UserIDs []uint64 `json:"user_ids" binding:"required"`
	}

	var req AssignUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.taskService.AssignUsers(services.AssignUsersInput{
		TaskID:  task.ID,
		ActorID: userID,
		UserIDs: req.UserIDs,
	}); err != nil {
		respondTaskError(c, err, "Failed to assign users")
		return
	}

	updatedTask, err := h.taskService.GetTask(task.ID)
	if err != nil {
		respondTaskError(c, err, "Failed to load task assignments")
		return
	}

	taskDTO := dto.ToTaskDTO(*updatedTask)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Users assigned successfully",
		"assignments": taskDTO.Assignments,
	})
}

// UnassignTask removes users from a task.
func (h *TaskHandler) UnassignTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	type UnassignUsersRequest struct {
		UserIDs []uint64 `json:"user_ids" binding:"required"`
	}

	var req UnassignUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	if err := h.taskService.UnassignUsers(task.ID, userID, req.UserIDs); err != nil {
		respondTaskError(c, err, "Failed to unassign users")
		return
	}

	updatedTask, err := h.taskService.GetTask(task.ID)
	if err != nil {
		respondTaskError(c, err, "Failed to load task assignments")
		return
	}

	taskDTO := dto.ToTaskDTO(*updatedTask)
	c.JSON(http.StatusOK, gin.H{
		"message":     "Users unassigned successfully",
		"assignments": taskDTO.Assignments,
	})
}

// GenerateTasks generates tasks via AI.
func (h *TaskHandler) GenerateTasks(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	type GenerateTasksRequest struct {
		Text           string `json:"text" binding:"required"`
		OrganizationID uint64 `json:"organization_id" binding:"required"`
	}

	var req GenerateTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.AIRequestTimeout)
	defer cancel()

	tasks, err := h.taskService.GenerateTasks(ctx, services.GenerateTasksInput{
		Text:           req.Text,
		OrganizationID: req.OrganizationID,
		CreatorID:      userID,
	})
	if err != nil {
		respondTaskError(c, err, "Failed to generate tasks")
		return
	}

	taskDTOs := make([]dto.TaskDTO, len(tasks))
	for i, task := range tasks {
		taskDTOs[i] = dto.ToTaskDTO(task)
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": taskDTOs,
	})
}

// ToggleTaskStatus toggles the task status between TODO and DONE.
func (h *TaskHandler) ToggleTaskStatus(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	task, ok := getTaskFromContext(c)
	if !ok {
		apierrors.InternalError(c, "Task not found in context")
		return
	}

	updatedTask, err := h.taskService.ToggleTaskStatus(task.ID, userID)
	if err != nil {
		respondTaskError(c, err, "Failed to toggle task status")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task status updated successfully",
		"status":  updatedTask.Status,
	})
}

// getTaskFromContext retrieves the task stored by middleware.
func getTaskFromContext(c *gin.Context) (models.Task, bool) {
	taskInterface, exists := c.Get(constants.ContextKeyTask)
	if !exists {
		return models.Task{}, false
	}

	task, ok := taskInterface.(models.Task)
	return task, ok
}

// respondTaskError maps domain errors to API responses.
func respondTaskError(c *gin.Context, err error, defaultMessage string) {
	switch {
	case stdErrors.Is(err, services.ErrNotOrganizationMember):
		apierrors.Forbidden(c, err.Error())
	case stdErrors.Is(err, services.ErrTaskNotFound):
		apierrors.NotFound(c, err.Error())
	case stdErrors.Is(err, services.ErrNotTaskCreator):
		apierrors.Forbidden(c, err.Error())
	case stdErrors.Is(err, services.ErrTaskPermissionDenied):
		apierrors.Forbidden(c, err.Error())
	case stdErrors.Is(err, services.ErrTitleRequired),
		stdErrors.Is(err, services.ErrTitleEmpty),
		stdErrors.Is(err, services.ErrInvalidTaskAssignee),
		stdErrors.Is(err, services.ErrNoUserIDsProvided),
		stdErrors.Is(err, services.ErrAINoTasksGenerated),
		stdErrors.Is(err, services.ErrAINoValidTasks):
		apierrors.BadRequest(c, err.Error())
	case stdErrors.Is(err, services.ErrAIServiceNotConfigured):
		apierrors.ServiceUnavailable(c, err.Error())
	default:
		apierrors.InternalError(c, defaultMessage)
	}
}
