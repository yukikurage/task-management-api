package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/dto"
	apierrors "github.com/yukikurage/task-management-api/internal/errors"
	"github.com/yukikurage/task-management-api/internal/middleware"
	"github.com/yukikurage/task-management-api/internal/services"
)

// AuthHandler coordinates authentication-related HTTP handlers.
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Signup registers a new user.
func (h *AuthHandler) Signup(c *gin.Context) {
	type SignupRequest struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Password string `json:"password" binding:"required"`
	}

	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	user, err := h.authService.Signup(services.SignupInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		respondAuthError(c, err)
		return
	}

	userDTO := dto.ToUserDTO(*user)
	c.JSON(http.StatusCreated, userDTO)
}

// Login authenticates a user and initializes the session.
func (h *AuthHandler) Login(c *gin.Context) {
	type LoginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierrors.BadRequest(c, "Invalid request body")
		return
	}

	user, err := h.authService.Login(services.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		respondAuthError(c, err)
		return
	}

	session := sessions.Default(c)
	session.Set(constants.ContextKeyUserID, user.ID)
	if err := session.Save(); err != nil {
		apierrors.InternalError(c, "Failed to save session")
		return
	}

	userDTO := dto.ToUserDTO(*user)
	c.JSON(http.StatusOK, userDTO)
}

// Logout removes the authentication session.
func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		apierrors.InternalError(c, "Failed to logout")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser returns the authenticated user.
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		apierrors.Unauthorized(c, "Not authenticated")
		return
	}

	user, err := h.authService.GetUser(userID)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	userDTO := dto.ToUserDTO(*user)
	c.JSON(http.StatusOK, userDTO)
}

func respondAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrPasswordTooShort):
		apierrors.BadRequest(c, fmt.Sprintf("Password must be at least %d characters", constants.MinPasswordLength))
	case errors.Is(err, services.ErrUsernameTaken):
		apierrors.Conflict(c, err.Error())
	case errors.Is(err, services.ErrInvalidCredentials):
		apierrors.Unauthorized(c, err.Error())
	case errors.Is(err, services.ErrUserNotFound):
		apierrors.NotFound(c, err.Error())
	case errors.Is(err, services.ErrFailedToHashPassword),
		errors.Is(err, services.ErrFailedToCreateUser),
		errors.Is(err, services.ErrFailedToCreateOrg),
		errors.Is(err, services.ErrFailedToAddMember):
		apierrors.InternalError(c, err.Error())
	default:
		apierrors.InternalError(c, "Internal server error")
	}
}
