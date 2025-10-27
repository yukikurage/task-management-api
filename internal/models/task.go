package models

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID             uint64         `gorm:"primarykey" json:"id"`
	Title          string         `gorm:"not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	DueDate        *time.Time     `json:"due_date"`
	CreatorID      uint64         `gorm:"not null" json:"creator_id"`
	OrganizationID uint64         `gorm:"not null" json:"organization_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Creator      User             `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Organization Organization     `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Assignments  []TaskAssignment `gorm:"foreignKey:TaskID" json:"assignments,omitempty"`
}
