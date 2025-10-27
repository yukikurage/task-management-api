package database

import (
	"gorm.io/gorm"

	"github.com/yukikurage/task-management-api/internal/utils"
)

// Paginate applies pagination to a GORM query
func Paginate(params utils.PaginationParams) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(params.Offset).Limit(params.Limit)
	}
}
