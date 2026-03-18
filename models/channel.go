package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChannelModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:(-)" json:"id"`
	WorkspaceID    uuid.UUID `gorm:"type:uuid;not null;index"         json:"workspaceId"`
	ConversationID uuid.UUID `gorm:"type:uuid;not null;unique"        json:"conversationId"`
	CreatedBy      uuid.UUID `gorm:"type:uuid;not null"               json:"createdBy"`
	Name           string    `gorm:"type:varchar(100);not null"       json:"name"`
	Description    *string   `gorm:"type:text"                        json:"description,omitempty"`
	IsPrivate      bool      `gorm:"not null;default:false"           json:"isPrivate"`
	CreatedAt      time.Time `                                        json:"createdAt"`
	UpdatedAt      time.Time `                                        json:"updatedAt"`

	// Relation to conversation for accessing participants and messages
	Conversation *ConversationModel `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
}

func (c *ChannelModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (ChannelModel) TableName() string {
	return "channels"
}
