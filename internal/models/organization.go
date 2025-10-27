package models

import (
	"time"

	"gorm.io/gorm"
)

type Organization struct {
	ID         uint64         `gorm:"primarykey" json:"id"`
	Name       string         `gorm:"type:varchar(255);not null" json:"name"`
	InviteCode string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"invite_code"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Members []OrganizationMember `gorm:"foreignKey:OrganizationID" json:"members,omitempty"`
	Tasks   []Task               `gorm:"foreignKey:OrganizationID" json:"tasks,omitempty"`
}
