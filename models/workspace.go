package models

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleGuest  Role = "guest"
)

type WorkspaceModel struct {
	Base
	Name        string    `gorm:"type:varchar(255);not null"       json:"name"`
	Description *string   `gorm:"type:text"                        json:"description,omitempty"`
	OwnerID     uuid.UUID `gorm:"type:uuid;not null;index"         json:"ownerId"`
}

type WorkspaceMemberModel struct {
	WorkspaceID uuid.UUID   `gorm:"type:uuid;primaryKey;index"  json:"workspaceId"`
	UserID      uuid.UUID   `gorm:"type:uuid;primaryKey;index"  json:"userId"`
	Role        Role        `gorm:"type:varchar(50);not null"   json:"role"`
	JoinedAt    time.Time   `                                   json:"joinedAt"`
	User        *UsersModel `gorm:"foreignKey:UserID;references:ID"  json:"user,omitempty"`
}

type WorkspaceInviteModel struct {
	Base
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index"         json:"workspaceId"`
	InvitedBy   uuid.UUID `gorm:"type:uuid;not null"`
	Email       string    `gorm:"type:varchar(255);not null;index" json:"email"`
	Role        Role      `gorm:"type:varchar(50);not null"        json:"role"`
	Token       string    `gorm:"type:varchar(64);not null;index"  json:"-"`
	Used        bool      `gorm:"not null;default:false"           json:"-"`
	ExpiresAt   time.Time `gorm:"not null"                         json:"expiresAt"`
}

func (WorkspaceModel) TableName() string {
	return "workspaces"
}

func (WorkspaceMemberModel) TableName() string {
	return "workspace_members"
}
func (WorkspaceInviteModel) TableName() string {
	return "workspace_invites"
}
