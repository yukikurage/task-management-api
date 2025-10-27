package models

import (
	"time"

	"gorm.io/gorm"
)

type TaskAssignment struct {
	TaskID    uint64         `gorm:"primarykey" json:"task_id"`
	UserID    uint64         `gorm:"primarykey" json:"user_id"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Task Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
