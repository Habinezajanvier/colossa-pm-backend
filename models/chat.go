package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationType string

const (
	ConversationTypeDM      ConversationType = "dm"
	ConversationTypeChannel ConversationType = "channel"
)

type ConversationModel struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey;default:(-)" json:"id"`
	WorkspaceID uuid.UUID        `gorm:"type:uuid;not null;index"         json:"workspaceId"`
	Type        ConversationType `gorm:"type:varchar(10);not null"        json:"type"`
	CreatedAt   time.Time        `                                        json:"createdAt"`

	// Relations
	Participants []ConversationParticipantModel `gorm:"foreignKey:ConversationID" json:"participants,omitempty"`
}

type ConversationParticipantModel struct {
	ConversationID uuid.UUID   `gorm:"type:uuid;primaryKey;index"      json:"conversationId"`
	UserID         uuid.UUID   `gorm:"type:uuid;primaryKey;index"      json:"userId"`
	IsAdmin        bool        `gorm:"not null;default:false"          json:"isAdmin"`
	JoinedAt       time.Time   `                                       json:"joinedAt"`
	User           *UsersModel `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

func (ConversationParticipantModel) TableName() string {
	return "conversation_participants"
}

// DMConversation wraps a conversation with all other participants populated
// Works for both 1-on-1 and group DMs
type DMConversation struct {
	ConversationModel
	OtherParticipants []UsersModel `json:"otherParticipants"`
}

func (c *ConversationModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (ConversationModel) TableName() string {
	return "conversations"
}

type ChatMessageModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:(-)"        json:"id"`
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index"                json:"conversationId"`
	SenderID       uuid.UUID  `gorm:"type:uuid;not null;index"                json:"senderId"`
	ParentID       *uuid.UUID `gorm:"type:uuid;index"                         json:"parentId,omitempty"`
	Body           *string    `gorm:"type:text"                               json:"body,omitempty"`
	DeletedAt      *time.Time `gorm:"index"                                   json:"deletedAt,omitempty"`
	CreatedAt      time.Time  `                                               json:"createdAt"`
	UpdatedAt      time.Time  `                                               json:"updatedAt"`

	// Relations
	Sender    *UsersModel            `gorm:"foreignKey:SenderID;references:ID"       json:"sender,omitempty"`
	Parent    *ChatMessageModel      `gorm:"foreignKey:ParentID;references:ID"       json:"parent,omitempty"`
	Files     []AttachmentModel      `gorm:"-"                                json:"files,omitempty"`
	Reactions []MessageReactionModel `gorm:"foreignKey:MessageID"                    json:"reactions,omitempty"`
	Replies   []ChatMessageModel     `gorm:"foreignKey:ParentID"                     json:"replies,omitempty"`
}

func (m *ChatMessageModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

func (ChatMessageModel) TableName() string {
	return "chat_messages"
}

type MessageReactionModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:(-)" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;index"         json:"messageId"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"         json:"userId"`
	Emoji     string    `gorm:"type:varchar(10);not null"        json:"emoji"`
	CreatedAt time.Time `                                        json:"createdAt"`

	// Relations
	User *UsersModel `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

func (r *MessageReactionModel) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (MessageReactionModel) TableName() string {
	return "message_reactions"
}
