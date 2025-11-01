package database

import (
	"fmt"

	"gorm.io/gorm"
)

// AddIndexes adds performance-critical indexes to the database
func AddIndexes(db *gorm.DB) error {
	// Tasks table indexes
	indexes := []struct {
		table string
		name  string
		columns string
	}{
		// Task indexes for filtering and sorting
		{"tasks", "idx_tasks_organization_id", "organization_id"},
		{"tasks", "idx_tasks_creator_id", "creator_id"},
		{"tasks", "idx_tasks_status", "status"},
		{"tasks", "idx_tasks_due_date", "due_date"},
		{"tasks", "idx_tasks_created_at", "created_at"},

		// Organization members indexes
		{"organization_members", "idx_org_members_organization_id", "organization_id"},
		{"organization_members", "idx_org_members_user_id", "user_id"},

		// Task assignments indexes
		{"task_assignments", "idx_task_assignments_task_id", "task_id"},
		{"task_assignments", "idx_task_assignments_user_id", "user_id"},

		// Organization invite code index
		{"organizations", "idx_organizations_invite_code", "invite_code"},
	}

	for _, idx := range indexes {
		// Check if index already exists
		var count int64
		err := db.Raw(`
			SELECT COUNT(*)
			FROM pg_indexes
			WHERE tablename = ? AND indexname = ?
		`, idx.table, idx.name).Count(&count).Error

		if err != nil {
			return fmt.Errorf("failed to check index %s: %w", idx.name, err)
		}

		if count > 0 {
			fmt.Printf("Index %s already exists, skipping\n", idx.name)
			continue
		}

		// Create index
		sql := fmt.Sprintf("CREATE INDEX %s ON %s (%s)", idx.name, idx.table, idx.columns)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}

		fmt.Printf("Created index %s on %s(%s)\n", idx.name, idx.table, idx.columns)
	}

	return nil
}

// MigrateDatabase runs all database migrations
func MigrateDatabase(db *gorm.DB) error {
	// Auto-migrate models (already done in InitDatabase)

	// Add indexes
	if err := AddIndexes(db); err != nil {
		return fmt.Errorf("failed to add indexes: %w", err)
	}

	return nil
}
