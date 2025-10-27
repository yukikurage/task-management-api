package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TaskHandlerTestSuite defines the test suite for TaskHandler
type TaskHandlerTestSuite struct {
	suite.Suite
	db      *gorm.DB
	handler *TaskHandler
	router  *gin.Engine
}

// SetupTest runs before each test
func (suite *TaskHandlerTestSuite) SetupTest() {
	var err error

	// Create in-memory SQLite database
	suite.db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Run migrations
	err = suite.db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Task{},
		&models.TaskAssignment{},
	)
	suite.Require().NoError(err)

	// Set the test DB as the default database
	database.SetDB(suite.db)

	// Create handler (without AI service for tests)
	suite.handler = NewTaskHandler(nil)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create router
	suite.router = gin.New()
}

// TearDownTest runs after each test
func (suite *TaskHandlerTestSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	suite.Require().NoError(err)
	sqlDB.Close()
}

// Helper function to create test data
func (suite *TaskHandlerTestSuite) createTestUser(email string) *models.User {
	user := &models.User{
		Email:        email,
		PasswordHash: "hashedpassword",
	}
	suite.db.Create(user)
	return user
}

func (suite *TaskHandlerTestSuite) createTestOrganization(name string) *models.Organization {
	org := &models.Organization{
		Name:       name,
		InviteCode: name + "_CODE",
	}
	suite.db.Create(org)
	return org
}

func (suite *TaskHandlerTestSuite) createTestOrganizationMember(orgID, userID uint64) *models.OrganizationMember {
	member := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
	}
	suite.db.Create(member)
	return member
}

func (suite *TaskHandlerTestSuite) createTestTask(title string, creatorID, orgID uint64) *models.Task {
	task := &models.Task{
		Title:          title,
		Description:    "Test Description",
		CreatorID:      creatorID,
		OrganizationID: orgID,
	}
	suite.db.Create(task)
	return task
}

// Helper function to create authenticated context
func (suite *TaskHandlerTestSuite) createAuthContext(method, url string, body []byte, userID uint64) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", userID)

	return c, w
}

// Helper function to set task context (simulates RequireTaskAccess middleware)
func (suite *TaskHandlerTestSuite) setTaskContext(c *gin.Context, task models.Task) {
	c.Set("task", task)
}

// TestListTasks_Success tests successful task listing
func (suite *TaskHandlerTestSuite) TestListTasks_Success() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user.ID)
	task := suite.createTestTask("Test Task", user.ID, org.ID)

	c, w := suite.createAuthContext("GET", "/api/tasks", nil, user.ID)
	c.Request.URL.RawQuery = "organization_id=1"

	suite.handler.ListTasks(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response, "tasks")
	assert.Contains(suite.T(), response, "pagination")

	tasks := response["tasks"].([]interface{})
	assert.GreaterOrEqual(suite.T(), len(tasks), 1)

	// Verify task is in the list
	firstTask := tasks[0].(map[string]interface{})
	assert.Equal(suite.T(), task.Title, firstTask["title"])
}

// TestListTasks_Unauthorized tests listing without authentication
func (suite *TaskHandlerTestSuite) TestListTasks_Unauthorized() {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/tasks", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	suite.handler.ListTasks(c)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

// TestListTasks_NotOrganizationMember tests listing when user is not a member
func (suite *TaskHandlerTestSuite) TestListTasks_NotOrganizationMember() {
	user := suite.createTestUser("test@example.com")
	suite.createTestOrganization("Test Org")
	// Don't add user as member

	c, w := suite.createAuthContext("GET", "/api/tasks", nil, user.ID)
	c.Request.URL.RawQuery = "organization_id=1"

	suite.handler.ListTasks(c)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestGetTask_Success tests successful task retrieval
func (suite *TaskHandlerTestSuite) TestGetTask_Success() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Test Task", user.ID, org.ID)

	// Reload task with relations
	suite.db.Preload("Creator").Preload("Organization").Preload("Assignments").First(&task, task.ID)

	c, w := suite.createAuthContext("GET", "/api/tasks/1", nil, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.GetTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.Task
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), task.ID, response.ID)
	assert.Equal(suite.T(), task.Title, response.Title)
}

// TestGetTask_NotFoundInContext tests when task is not in context
func (suite *TaskHandlerTestSuite) TestGetTask_NotFoundInContext() {
	user := suite.createTestUser("test@example.com")
	c, w := suite.createAuthContext("GET", "/api/tasks/1", nil, user.ID)

	suite.handler.GetTask(c)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
}

// TestCreateTask_Success tests successful task creation
func (suite *TaskHandlerTestSuite) TestCreateTask_Success() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user.ID)

	requestBody := map[string]interface{}{
		"title":           "New Task",
		"description":     "Task Description",
		"organization_id": org.ID,
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks", body, user.ID)

	suite.handler.CreateTask(c)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response models.Task
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "New Task", response.Title)
	assert.Equal(suite.T(), user.ID, response.CreatorID)
}

// TestCreateTask_InvalidRequest tests task creation with invalid request
func (suite *TaskHandlerTestSuite) TestCreateTask_InvalidRequest() {
	user := suite.createTestUser("test@example.com")

	// Missing required field: title
	requestBody := map[string]interface{}{
		"organization_id": 1,
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks", body, user.ID)

	suite.handler.CreateTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestCreateTask_NotOrganizationMember tests task creation when user is not a member
func (suite *TaskHandlerTestSuite) TestCreateTask_NotOrganizationMember() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	// Don't add user as member

	requestBody := map[string]interface{}{
		"title":           "New Task",
		"description":     "Task Description",
		"organization_id": org.ID,
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks", body, user.ID)

	suite.handler.CreateTask(c)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestUpdateTask_Success tests successful task update
func (suite *TaskHandlerTestSuite) TestUpdateTask_Success() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Old Title", user.ID, org.ID)

	requestBody := map[string]interface{}{
		"title":       "Updated Title",
		"description": "Updated Description",
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("PUT", "/api/tasks/1", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UpdateTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.Task
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Title", response.Title)
	assert.Equal(suite.T(), "Updated Description", response.Description)
}

// TestUpdateTask_NullDueDate tests updating due_date to null
func (suite *TaskHandlerTestSuite) TestUpdateTask_NullDueDate() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	dueDate := time.Now().Add(24 * time.Hour)
	task := suite.createTestTask("Task with Due Date", user.ID, org.ID)
	task.DueDate = &dueDate
	suite.db.Save(task)

	requestBody := map[string]interface{}{
		"due_date": nil,
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("PUT", "/api/tasks/1", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UpdateTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response models.Task
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Nil(suite.T(), response.DueDate)
}

// TestUpdateTask_InvalidRequest tests task update with invalid request
func (suite *TaskHandlerTestSuite) TestUpdateTask_InvalidRequest() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Test Task", user.ID, org.ID)

	c, w := suite.createAuthContext("PUT", "/api/tasks/1", []byte("invalid json"), user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UpdateTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestDeleteTask_Success tests successful task deletion
func (suite *TaskHandlerTestSuite) TestDeleteTask_Success() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Delete", user.ID, org.ID)

	c, w := suite.createAuthContext("DELETE", "/api/tasks/1", nil, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.DeleteTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Task deleted successfully", response["message"])

	// Verify task is deleted
	var deletedTask models.Task
	err = suite.db.First(&deletedTask, task.ID).Error
	assert.Error(suite.T(), err) // Should return error because of soft delete
}

// TestDeleteTask_NotCreator tests task deletion by non-creator
func (suite *TaskHandlerTestSuite) TestDeleteTask_NotCreator() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Delete", user1.ID, org.ID)

	c, w := suite.createAuthContext("DELETE", "/api/tasks/1", nil, user2.ID)
	suite.setTaskContext(c, *task)

	suite.handler.DeleteTask(c)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestAssignTask_Success tests successful task assignment
func (suite *TaskHandlerTestSuite) TestAssignTask_Success() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user1.ID)
	suite.createTestOrganizationMember(org.ID, user2.ID)
	task := suite.createTestTask("Task to Assign", user1.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user2.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/assign", body, user1.ID)
	suite.setTaskContext(c, *task)

	suite.handler.AssignTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Users assigned successfully", response["message"])

	// Verify assignment was created
	var assignment models.TaskAssignment
	err = suite.db.Where("task_id = ? AND user_id = ?", task.ID, user2.ID).First(&assignment).Error
	assert.NoError(suite.T(), err)
}

// TestAssignTask_NotCreator tests task assignment by non-creator
func (suite *TaskHandlerTestSuite) TestAssignTask_NotCreator() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	user3 := suite.createTestUser("user3@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Assign", user1.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user3.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/assign", body, user2.ID)
	suite.setTaskContext(c, *task)

	suite.handler.AssignTask(c)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestAssignTask_EmptyUserIDs tests task assignment with empty user IDs
func (suite *TaskHandlerTestSuite) TestAssignTask_EmptyUserIDs() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Assign", user.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/assign", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.AssignTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestAssignTask_UserNotExists tests task assignment with non-existent user
func (suite *TaskHandlerTestSuite) TestAssignTask_UserNotExists() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Assign", user.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{9999}, // Non-existent user
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/assign", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.AssignTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestAssignTask_UserNotOrganizationMember tests task assignment with user not in organization
func (suite *TaskHandlerTestSuite) TestAssignTask_UserNotOrganizationMember() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user1.ID)
	// Don't add user2 to organization
	task := suite.createTestTask("Task to Assign", user1.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user2.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/assign", body, user1.ID)
	suite.setTaskContext(c, *task)

	suite.handler.AssignTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUnassignTask_Success tests successful task unassignment
func (suite *TaskHandlerTestSuite) TestUnassignTask_Success() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user1.ID)
	suite.createTestOrganizationMember(org.ID, user2.ID)
	task := suite.createTestTask("Task to Unassign", user1.ID, org.ID)

	// Create assignment first
	assignment := &models.TaskAssignment{
		TaskID: task.ID,
		UserID: user2.ID,
	}
	suite.db.Create(assignment)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user2.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/unassign", body, user1.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UnassignTask(c)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Users unassigned successfully", response["message"])

	// Verify assignment was deleted
	var deletedAssignment models.TaskAssignment
	err = suite.db.Where("task_id = ? AND user_id = ?", task.ID, user2.ID).First(&deletedAssignment).Error
	assert.Error(suite.T(), err) // Should return error because assignment is deleted
}

// TestUnassignTask_NotCreator tests task unassignment by non-creator
func (suite *TaskHandlerTestSuite) TestUnassignTask_NotCreator() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Unassign", user1.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user2.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/unassign", body, user2.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UnassignTask(c)

	assert.Equal(suite.T(), http.StatusForbidden, w.Code)
}

// TestUnassignTask_EmptyUserIDs tests task unassignment with empty user IDs
func (suite *TaskHandlerTestSuite) TestUnassignTask_EmptyUserIDs() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Unassign", user.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/unassign", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UnassignTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUnassignTask_UserNotExists tests task unassignment with non-existent user
func (suite *TaskHandlerTestSuite) TestUnassignTask_UserNotExists() {
	user := suite.createTestUser("test@example.com")
	org := suite.createTestOrganization("Test Org")
	task := suite.createTestTask("Task to Unassign", user.ID, org.ID)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{9999}, // Non-existent user
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/unassign", body, user.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UnassignTask(c)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)
}

// TestUnassignTask_OrganizationExternalUser tests unassigning user not in organization (allowed after RemoveUser)
func (suite *TaskHandlerTestSuite) TestUnassignTask_OrganizationExternalUser() {
	user1 := suite.createTestUser("user1@example.com")
	user2 := suite.createTestUser("user2@example.com")
	org := suite.createTestOrganization("Test Org")
	suite.createTestOrganizationMember(org.ID, user1.ID)
	task := suite.createTestTask("Task to Unassign", user1.ID, org.ID)

	// Create assignment (simulating user was previously in org)
	assignment := &models.TaskAssignment{
		TaskID: task.ID,
		UserID: user2.ID,
	}
	suite.db.Create(assignment)

	requestBody := map[string]interface{}{
		"user_ids": []uint64{user2.ID},
	}
	body, _ := json.Marshal(requestBody)

	c, w := suite.createAuthContext("POST", "/api/tasks/1/unassign", body, user1.ID)
	suite.setTaskContext(c, *task)

	suite.handler.UnassignTask(c)

	// Should succeed - allows cleanup after user removed from org
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Users unassigned successfully", response["message"])
}

// TestSuite runs the test suite
func TestTaskHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TaskHandlerTestSuite))
}
