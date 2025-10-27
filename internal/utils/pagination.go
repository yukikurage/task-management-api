package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}
