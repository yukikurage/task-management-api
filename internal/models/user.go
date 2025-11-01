package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint64         `gorm:"primarykey" json:"id"`
	Username     string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	CreatedTasks []Task                 `gorm:"foreignKey:CreatorID" json:"-"`
	Assignments  []TaskAssignment       `gorm:"foreignKey:UserID" json:"-"`
	Organizations []OrganizationMember  `gorm:"foreignKey:UserID" json:"-"`
}
