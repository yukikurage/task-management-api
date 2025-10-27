package models

import "time"

type OrganizationRole string

const (
	RoleOwner  OrganizationRole = "owner"
	RoleMember OrganizationRole = "member"
)

type OrganizationMember struct {
	OrganizationID uint64           `gorm:"primarykey" json:"organization_id"`
	UserID         uint64           `gorm:"primarykey" json:"user_id"`
	Role           OrganizationRole `gorm:"type:varchar(20);not null" json:"role"`
	JoinedAt       time.Time        `json:"joined_at"`

	// Relations
	Organization Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
