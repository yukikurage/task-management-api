package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yukikurage/task-management-api/internal/constants"
	"github.com/yukikurage/task-management-api/internal/models"
	"github.com/yukikurage/task-management-api/internal/repository"
	"github.com/yukikurage/task-management-api/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUsernameTaken        = errors.New("username already exists")
	ErrInvalidCredentials   = errors.New("invalid username or password")
	ErrPasswordTooShort     = errors.New("password too short")
	ErrUserNotFound         = errors.New("user not found")
	ErrFailedToHashPassword = errors.New("failed to hash password")
	ErrFailedToCreateUser   = errors.New("failed to create user")
	ErrFailedToCreateOrg    = errors.New("failed to create organization")
	ErrFailedToAddMember    = errors.New("failed to add user to organization")
)

// AuthService handles authentication related business logic.
type AuthService struct {
	userRepo repository.UserRepository
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

// SignupInput represents the required information to create a new user.
type SignupInput struct {
	Username string
	Password string
}

// Signup creates a new user along with a personal organization.
func (s *AuthService) Signup(input SignupInput) (*models.User, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len(input.Password) < constants.MinPasswordLength {
		return nil, ErrPasswordTooShort
	}

	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrFailedToHashPassword
	}

	user := &models.User{
		Username:     username,
		PasswordHash: string(hashedPassword),
	}

	orgName := fmt.Sprintf("%sの組織", user.Username)
	inviteCode, err := utils.GenerateInviteCode()
	if err != nil {
		return nil, ErrFailedToCreateOrg
	}

	org := &models.Organization{
		Name:       orgName,
		InviteCode: inviteCode,
	}

	member := &models.OrganizationMember{
		Role:           models.RoleOwner,
		JoinedAt:       time.Now(),
	}

	if err := s.userRepo.CreateWithPersonalOrganization(user, org, member); err != nil {
		switch {
		case errors.Is(err, repository.ErrCreateUser):
			return nil, ErrFailedToCreateUser
		case errors.Is(err, repository.ErrCreateOrganization):
			return nil, ErrFailedToCreateOrg
		case errors.Is(err, repository.ErrCreateOrganizationMember):
			return nil, ErrFailedToAddMember
		default:
			return nil, fmt.Errorf("failed to complete signup: %w", err)
		}
	}

	return user, nil
}

// LoginInput holds the credentials for authentication.
type LoginInput struct {
	Username string
	Password string
}

// Login verifies credentials and returns the authenticated user.
func (s *AuthService) Login(input LoginInput) (*models.User, error) {
	user, err := s.userRepo.FindByUsername(input.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// GetUser retrieves a user by ID.
func (s *AuthService) GetUser(id uint64) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return user, nil
}
