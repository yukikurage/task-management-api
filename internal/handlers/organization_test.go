package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

type organizationTestEnv struct {
	db         *gorm.DB
	handler    *OrganizationHandler
	orgService *services.OrganizationService
}

func setupOrganizationTestEnv(t *testing.T) organizationTestEnv {
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

	orgRepo := repository.NewOrganizationRepository(db)
	orgService := services.NewOrganizationService(orgRepo)
	handler := NewOrganizationHandler(orgService)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	return organizationTestEnv{
		db:         db,
		handler:    handler,
		orgService: orgService,
	}
}

func orgTestContext(method, url string, body []byte, userID uint64) (*gin.Context, *httptest.ResponseRecorder) {
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

func createTestOrganizationUser(t *testing.T, db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username:     username,
		PasswordHash: "hashed",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func TestOrganizationHandler_CreateOrganization(t *testing.T) {
	env := setupOrganizationTestEnv(t)

	user := createTestOrganizationUser(t, env.db, "owner")

	payload := map[string]string{"name": "New Org"}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	c, w := orgTestContext(http.MethodPost, "/api/organizations", body, user.ID)

	env.handler.CreateOrganization(c)

	require.Equal(t, http.StatusCreated, w.Code)

	var response dto.OrganizationDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, payload["name"], response.Name)
	require.NotEmpty(t, response.InviteCode)
}

func TestOrganizationHandler_ListOrganizations(t *testing.T) {
	env := setupOrganizationTestEnv(t)

	user := createTestOrganizationUser(t, env.db, "member")

	_, err := env.orgService.CreateOrganization(services.CreateOrganizationInput{
		Name:    "Org One",
		OwnerID: user.ID,
	})
	require.NoError(t, err)

	c, w := orgTestContext(http.MethodGet, "/api/organizations", nil, user.ID)

	env.handler.ListOrganizations(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response map[string][]dto.OrganizationWithRoleDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	orgs := response["organizations"]
	require.Len(t, orgs, 1)
	require.Equal(t, "Org One", orgs[0].OrganizationDTO.Name)
	require.Equal(t, models.RoleOwner, orgs[0].Role)
}

func TestOrganizationHandler_JoinOrganization_InvalidCode(t *testing.T) {
	env := setupOrganizationTestEnv(t)

	user := createTestOrganizationUser(t, env.db, "user")

	payload := map[string]string{"invite_code": "UNKNOWN"}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	c, w := orgTestContext(http.MethodPost, "/api/organizations/join", body, user.ID)

	env.handler.JoinOrganization(c)

	require.Equal(t, http.StatusNotFound, w.Code)
}
