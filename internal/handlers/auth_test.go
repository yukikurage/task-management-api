package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
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

type authTestEnv struct {
	db          *gorm.DB
	handler     *AuthHandler
	authService *services.AuthService
}

func setupAuthTestEnv(t *testing.T) authTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrganizationMember{},
	)
	require.NoError(t, err)

	database.SetDB(db)

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo)
	handler := NewAuthHandler(authService)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB.Close()
	})

	return authTestEnv{
		db:          db,
		handler:     handler,
		authService: authService,
	}
}

func TestAuthHandler_Signup(t *testing.T) {
	env := setupAuthTestEnv(t)

	r := gin.New()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions(constants.SessionCookieName, store))
	r.POST("/api/auth/signup", env.handler.Signup)

	payload := map[string]string{
		"username": "newuser",
		"password": "supersecret",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response dto.UserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, payload["username"], response.Username)
}

func TestAuthHandler_Login(t *testing.T) {
	env := setupAuthTestEnv(t)

	_, err := env.authService.Signup(services.SignupInput{
		Username: "existing",
		Password: "supersecret",
	})
	require.NoError(t, err)

	r := gin.New()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions(constants.SessionCookieName, store))
	r.POST("/api/auth/login", env.handler.Login)

	payload := map[string]string{
		"username": "existing",
		"password": "supersecret",
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response dto.UserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, payload["username"], response.Username)

	cookies := w.Result().Cookies()
	require.NotEmpty(t, cookies, "expected session cookie to be set")
}

func TestAuthHandler_GetCurrentUser(t *testing.T) {
	env := setupAuthTestEnv(t)

	user, err := env.authService.Signup(services.SignupInput{
		Username: "current-user",
		Password: "supersecret",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(constants.ContextKeyUserID, user.ID)

	env.handler.GetCurrentUser(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response dto.UserDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, user.Username, response.Username)
}
