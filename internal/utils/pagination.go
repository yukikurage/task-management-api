package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yukikurage/task-management-api/internal/constants"
)

// PaginationParams holds the pagination parameters
type PaginationParams struct {
	Page   int
	Limit  int
	Offset int
}

// PaginationResponse represents the pagination metadata in API responses
type PaginationResponse struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// GetPaginationParams extracts and validates pagination parameters from the request
func GetPaginationParams(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(constants.MinPageSize)))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(constants.DefaultPageSize)))

	if page < constants.MinPageSize {
		page = constants.MinPageSize
	}
	if limit < constants.MinPageSize || limit > constants.MaxPageSize {
		limit = constants.DefaultPageSize
	}

	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}
