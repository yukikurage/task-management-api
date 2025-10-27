package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/services"
	"github.com/yukikurage/task-management-api/internal/utils"
	"gorm.io/gorm/clause"
)

type TaskHandler struct {
	aiService *services.AIService
}

func NewTaskHandler(aiService *services.AIService) *TaskHandler {
	return &TaskHandler{
		aiService: aiService,
	}
}

// ListTasks returns all tasks accessible by the current user
// Can filter by organization_id
func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get organization filter (optional)
	organizationIDStr := c.Query("organization_id")
	var organizationID uint64
	if organizationIDStr != "" {
		orgID, err := strconv.ParseUint(organizationIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization_id"})
			return
		}
		organizationID = orgID

		// Verify user is a member of this organization
		var member models.OrganizationMember
		if err := database.GetDB().
			Where("organization_id = ? AND user_id = ?", organizationID, userID).
			First(&member).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this organization"})
			return
		}
	}

	// Get pagination parameters
	params := utils.GetPaginationParams(c)

	// Get organization IDs where user is a member
	var orgIDs []uint
	database.GetDB().
		Model(&models.OrganizationMember{}).
		Where("user_id = ?", userID).
		Pluck("organization_id", &orgIDs)

	// Query tasks in user's organizations
	var tasks []models.Task
	query := database.GetDB().
		Preload("Creator").
		Preload("Organization").
		Preload("Assignments").
		Preload("Assignments.User")

	if organizationID != 0 {
		// Filter by specific organization
		query = query.Where("organization_id = ?", organizationID)
	} else {
		// Get tasks from all user's organizations
		if len(orgIDs) > 0 {
			query = query.Where("organization_id IN ?", orgIDs)
		} else {
			// User has no organizations, return empty
			c.JSON(http.StatusOK, gin.H{
				"tasks": []models.Task{},
				"pagination": utils.PaginationResponse{
					Page:  params.Page,
					Limit: params.Limit,
					Total: 0,
				},
			})
			return
		}
	}

	// Get total count
	var total int64
	query.Model(&models.Task{}).Count(&total)

	// Get paginated results
	if err := query.Scopes(database.Paginate(params)).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"pagination": utils.PaginationResponse{
			Page:  params.Page,
			Limit: params.Limit,
			Total: total,
		},
	})
}

// GetTask returns a specific task by ID
// Task is already loaded with relations by RequireTaskAccess middleware
func (h *TaskHandler) GetTask(c *gin.Context) {
	// Get task from context (set by RequireTaskAccess middleware)
	taskInterface, exists := c.Get("task")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task not found in context"})
		return
	}

	task, ok := taskInterface.(models.Task)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid task data"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type CreateTaskRequest struct {
		Title          string     `json:"title" binding:"required"`
		Description    string     `json:"description"`
		DueDate        *time.Time `json:"due_date"`
		OrganizationID uint64     `json:"organization_id" binding:"required"`
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify user is a member of the organization
	var member models.OrganizationMember
	if err := database.GetDB().
		Where("organization_id = ? AND user_id = ?", req.OrganizationID, userID).
		First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this organization"})
		return
	}

	// Create task
	task := models.Task{
		Title:          req.Title,
		Description:    req.Description,
		DueDate:        req.DueDate,
		CreatorID:      userID,
		OrganizationID: req.OrganizationID,
	}

	if err := database.GetDB().Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Load relations
	database.GetDB().
		Preload("Creator").
		Preload("Organization").
		First(&task, task.ID)

	c.JSON(http.StatusCreated, task)
}

// UpdateTask updates an existing task
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	taskInterface, exists := c.Get("task")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task not found in context"})
		return
	}

	task, ok := taskInterface.(models.Task)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid task data"})
		return
	}

	// Parse raw JSON to detect which fields were sent
	var rawReq map[string]any
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Update only provided fields
	if title, ok := rawReq["title"]; ok {
		if titleStr, ok := title.(string); ok {
			task.Title = titleStr
		}
	}
	if description, ok := rawReq["description"]; ok {
		if descStr, ok := description.(string); ok {
			task.Description = descStr
		}
	}
	if _, ok := rawReq["due_date"]; ok {
		// due_date was provided (might be null)
		if rawReq["due_date"] == nil {
			task.DueDate = nil
		} else if dueDateStr, ok := rawReq["due_date"].(string); ok {
			parsedTime, err := time.Parse(time.RFC3339, dueDateStr)
			if err == nil {
				task.DueDate = &parsedTime
			}
		}
	}

	// Save to database
	if err := database.GetDB().Save(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Reload with relations
	if err := database.GetDB().
		Preload("Creator").
		Preload("Organization").
		Preload("Assignments").
		Preload("Assignments.User").
		First(&task, task.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload task"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask deletes a task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	taskInterface, exists := c.Get("task")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task not found in context"})
		return
	}

	task, ok := taskInterface.(models.Task)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid task data"})
		return
	}

	// Only creator can delete
	if task.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the creator can delete this task"})
		return
	}

	// Delete task assignments first
	if err := database.GetDB().Where("task_id = ?", task.ID).Delete(&models.TaskAssignment{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task assignments"})
		return
	}

	if err := database.GetDB().Delete(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
	})
}

// AssignTask assigns users to a task
func (h *TaskHandler) AssignTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get current task
	taskInterface, exists := c.Get("task")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task not found in context"})
		return
	}

	task, ok := taskInterface.(models.Task)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid task data"})
		return
	}

	type AssignUserRequest struct {
		UserIDs []uint64 `json:"user_ids" binding:"required"`
	}

	// Only creator can assign
	if task.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the creator can assign this task"})
		return
	}

	// Get request
	var req AssignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if array is empty
	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one user ID is required"})
		return
	}

	// Verify all users exist
	var userCount int64
	database.GetDB().Model(&models.User{}).Where("id IN ?", req.UserIDs).Count(&userCount)
	if int(userCount) != len(req.UserIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "One or more user IDs do not exist"})
		return
	}

	// Verify all users are members of the organization
	var memberCount int64
	database.GetDB().
		Model(&models.OrganizationMember{}).
		Where("organization_id = ? AND user_id IN ?", task.OrganizationID, req.UserIDs).
		Count(&memberCount)

	if int(memberCount) != len(req.UserIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "One or more users are not members of this organization"})
		return
	}

	// Create task assignments
	var taskAssignments []models.TaskAssignment
	for _, uid := range req.UserIDs {
		taskAssignments = append(taskAssignments, models.TaskAssignment{
			TaskID: task.ID,
			UserID: uid,
		})
	}

	if err := database.GetDB().
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&taskAssignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign users to task"})
		return
	}

	// Reload task with assignments
	if err := database.GetDB().
		Preload("Assignments").
		Preload("Assignments.User").
		First(&task, task.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload task assignments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Users assigned successfully",
		"assignments": task.Assignments,
	})
}

// UnassignTask removes user assignments from a task
func (h *TaskHandler) UnassignTask(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Get current task
	taskInterface, exists := c.Get("task")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Task not found in context"})
		return
	}

	task, ok := taskInterface.(models.Task)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid task data"})
		return
	}

	type AssignUserRequest struct {
		UserIDs []uint64 `json:"user_ids" binding:"required"`
	}

	// Only creator can unassign
	if task.CreatorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the creator can unassign from this task"})
		return
	}

	// Get request
	var req AssignUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if array is empty
	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one user ID is required"})
		return
	}

	// Verify all users exist
	var userCount int64
	database.GetDB().Model(&models.User{}).Where("id IN ?", req.UserIDs).Count(&userCount)
	if int(userCount) != len(req.UserIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "One or more user IDs do not exist"})
		return
	}

	// Delete task assignments
	if err := database.GetDB().
		Where("task_id = ? AND user_id IN ?", task.ID, req.UserIDs).
		Delete(&models.TaskAssignment{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unassign users from task"})
		return
	}

	// Reload task with assignments
	if err := database.GetDB().
		Preload("Assignments").
		Preload("Assignments.User").
		First(&task, task.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload task assignments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Users unassigned successfully",
		"assignments": task.Assignments,
	})
}

// GenerateTasks generates task suggestions from text using AI
func (h *TaskHandler) GenerateTasks(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	type GenerateTasksRequest struct {
		Text           string `json:"text" binding:"required"`
		OrganizationID uint64 `json:"organization_id" binding:"required"`
	}

	var req GenerateTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify user is a member of the organization
	var member models.OrganizationMember
	if err := database.GetDB().
		Where("organization_id = ? AND user_id = ?", req.OrganizationID, userID).
		First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this organization"})
		return
	}

	// Check if AI service is available
	if h.aiService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service is not configured. Please set OPENAI_API_KEY environment variable."})
		return
	}

	// Generate tasks using AI
	ctx := context.Background()
	generatedTasks, err := h.aiService.GenerateTasksFromText(ctx, req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate tasks: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": generatedTasks,
	})
}
