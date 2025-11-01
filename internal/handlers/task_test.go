package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/database"
	"github.com/yukikurage/task-management-api/internal/dto"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/repository"
	"github.com/yukikurage/task-management-api/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type taskHandlerTestEnv struct {
	handler     *TaskHandler
	taskService *services.TaskService
	db          *gorm.DB
}

func setupTaskHandlerTestEnv(t *testing.T) taskHandlerTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Task{},
		&models.TaskAssignment{},
	)
	require.NoError(t, err)

	database.SetDB(db)

	taskRepo := repository.NewTaskRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)
	taskService := services.NewTaskService(taskRepo, orgRepo, nil)
	handler := NewTaskHandler(taskService)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	return taskHandlerTestEnv{
		handler:     handler,
		taskService: taskService,
		db:          db,
	}
}

func createUser(t *testing.T, db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username:     username,
		PasswordHash: "hashed",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func createOrganization(t *testing.T, db *gorm.DB, name string) *models.Organization {
	org := &models.Organization{
		Name:       name,
		InviteCode: name + "_CODE",
	}
	require.NoError(t, db.Create(org).Error)
	return org
}

func addMember(t *testing.T, db *gorm.DB, orgID, userID uint64) {
	member := &models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         userID,
	}
	require.NoError(t, db.Create(member).Error)
}

func newTestContext(method, url string, body []byte, userID uint64) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set(constants.ContextKeyUserID, userID)

	return c, w
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	env := setupTaskHandlerTestEnv(t)

	user := createUser(t, env.db, "creator")
	org := createOrganization(t, env.db, "Org")
	addMember(t, env.db, org.ID, user.ID)

	payload := map[string]any{
		"title":           "Write docs",
		"description":     "Documentation work",
		"organization_id": org.ID,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	c, w := newTestContext(http.MethodPost, "/api/tasks", body, user.ID)

	env.handler.CreateTask(c)

	require.Equal(t, http.StatusCreated, w.Code)

	var response dto.TaskDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, payload["title"], response.Title)
	require.Equal(t, org.ID, response.OrganizationID)
	require.Equal(t, user.ID, response.CreatorID)
	require.Len(t, response.Assignments, 1)
}

func TestTaskHandler_ListTasks_ByOrganization(t *testing.T) {
	env := setupTaskHandlerTestEnv(t)

	user := createUser(t, env.db, "member")
	org := createOrganization(t, env.db, "Org")
	addMember(t, env.db, org.ID, user.ID)

	_, err := env.taskService.CreateTask(services.CreateTaskInput{
		Title:          "Task A",
		Description:    "A",
		OrganizationID: org.ID,
		CreatorID:      user.ID,
	})
	require.NoError(t, err)

	c, w := newTestContext(http.MethodGet, "/api/tasks", nil, user.ID)
	q := c.Request.URL.Query()
	q.Set("organization_id", strconv.FormatUint(org.ID, 10))
	c.Request.URL.RawQuery = q.Encode()

	env.handler.ListTasks(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response dto.TaskListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, int64(1), response.TotalCount)
	require.Len(t, response.Tasks, 1)
	require.Equal(t, "Task A", response.Tasks[0].Title)
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	env := setupTaskHandlerTestEnv(t)

	user := createUser(t, env.db, "owner")
	org := createOrganization(t, env.db, "Org")
	addMember(t, env.db, org.ID, user.ID)

	createdTask, err := env.taskService.CreateTask(services.CreateTaskInput{
		Title:          "Task to remove",
		Description:    "Remove me",
		OrganizationID: org.ID,
		CreatorID:      user.ID,
	})
	require.NoError(t, err)

	c, w := newTestContext(http.MethodDelete, "/api/tasks/"+strconv.FormatUint(createdTask.ID, 10), nil, user.ID)
	c.Set(constants.ContextKeyTask, *createdTask)

	env.handler.DeleteTask(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Task deleted successfully", response["message"])

	var count int64
	require.NoError(t, env.db.Model(&models.Task{}).Where("id = ?", createdTask.ID).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestTaskHandler_ToggleTaskStatus_AssignedUser(t *testing.T) {
	env := setupTaskHandlerTestEnv(t)

	creator := createUser(t, env.db, "creator")
	assignee := createUser(t, env.db, "assignee")
	org := createOrganization(t, env.db, "Org")
	addMember(t, env.db, org.ID, creator.ID)
	addMember(t, env.db, org.ID, assignee.ID)

	task, err := env.taskService.CreateTask(services.CreateTaskInput{
		Title:          "Toggle Task",
		Description:    "Toggle description",
		OrganizationID: org.ID,
		CreatorID:      creator.ID,
	})
	require.NoError(t, err)

	err = env.taskService.AssignUsers(services.AssignUsersInput{
		TaskID:  task.ID,
		ActorID: creator.ID,
		UserIDs: []uint64{assignee.ID},
	})
	require.NoError(t, err)

	c, w := newTestContext(http.MethodPost, "/api/tasks/"+strconv.FormatUint(task.ID, 10)+"/toggle-status", nil, assignee.ID)
	c.Set(constants.ContextKeyTask, models.Task{
		ID:        task.ID,
		CreatorID: creator.ID,
		Status:    models.TaskStatusTodo,
	})

	env.handler.ToggleTaskStatus(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, string(models.TaskStatusDone), response["status"])
}
