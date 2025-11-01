package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Error codes
const (
	// Authentication errors
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"

	// Authorization errors
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"

	// Validation errors
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingField     = "MISSING_FIELD"
	ErrCodeInvalidFormat    = "INVALID_FORMAT"

	// Resource errors
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeAlreadyExists    = "ALREADY_EXISTS"
	ErrCodeConflict         = "CONFLICT"

	// Business logic errors
	ErrCodeInvalidOperation = "INVALID_OPERATION"
	ErrCodeOperationFailed  = "OPERATION_FAILED"

	// Service errors
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// APIError represents a standardized API error response
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return e.Message
}

// NewAPIError creates a new APIError
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewAPIErrorWithDetails creates a new APIError with details
func NewAPIErrorWithDetails(code, message string, details interface{}) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Predefined errors
var (
	ErrUnauthorized = NewAPIError(ErrCodeUnauthorized, "Authentication required")
	ErrForbidden    = NewAPIError(ErrCodeForbidden, "Access denied")
	ErrNotFound     = NewAPIError(ErrCodeNotFound, "Resource not found")
	ErrInvalidInput = NewAPIError(ErrCodeInvalidInput, "Invalid request body")
	ErrInternalError = NewAPIError(ErrCodeInternalError, "Internal server error")
	ErrServiceUnavailable = NewAPIError(ErrCodeServiceUnavailable, "Service temporarily unavailable")
)

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, statusCode int, err *APIError) {
	c.JSON(statusCode, err)
}

// Helper functions for common error responses

// Unauthorized sends a 401 response
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	RespondWithError(c, http.StatusUnauthorized, NewAPIError(ErrCodeUnauthorized, message))
}

// Forbidden sends a 403 response
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Access denied"
	}
	RespondWithError(c, http.StatusForbidden, NewAPIError(ErrCodeForbidden, message))
}

// NotFound sends a 404 response
func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Resource not found"
	}
	RespondWithError(c, http.StatusNotFound, NewAPIError(ErrCodeNotFound, message))
}

// BadRequest sends a 400 response
func BadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "Invalid request"
	}
	RespondWithError(c, http.StatusBadRequest, NewAPIError(ErrCodeInvalidInput, message))
}

// BadRequestWithDetails sends a 400 response with details
func BadRequestWithDetails(c *gin.Context, message string, details interface{}) {
	RespondWithError(c, http.StatusBadRequest, NewAPIErrorWithDetails(ErrCodeInvalidInput, message, details))
}

// Conflict sends a 409 response
func Conflict(c *gin.Context, message string) {
	if message == "" {
		message = "Resource conflict"
	}
	RespondWithError(c, http.StatusConflict, NewAPIError(ErrCodeConflict, message))
}

// InternalError sends a 500 response
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	RespondWithError(c, http.StatusInternalServerError, NewAPIError(ErrCodeInternalError, message))
}

// ServiceUnavailable sends a 503 response
func ServiceUnavailable(c *gin.Context, message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	RespondWithError(c, http.StatusServiceUnavailable, NewAPIError(ErrCodeServiceUnavailable, message))
}
