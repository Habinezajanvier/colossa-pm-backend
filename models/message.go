package models

import (
	"github.com/google/uuid"
)

type MessageStatus string
type MessageType string

type MessageModel struct {
	Base
	UserID    uuid.UUID     `gorm:"type:uuid;not null;index"         json:"userId"`
	Recipient string        `gorm:"type:varchar(255);not null"       json:"recipient"`
	Type      MessageType   `gorm:"type:varchar(50);not null;index"  json:"type"`
	Status    MessageStatus `gorm:"type:varchar(20);not null"         json:"status"`
	Error     *string       `gorm:"type:text"                        json:"error,omitempty"`
	EventType *string       `gorm:"type:varchar(100);index:idx_event_type_id,composite:1" json:"eventType,omitempty"`
	EventID   *uuid.UUID    `gorm:"type:uuid;index:idx_event_type_id,composite:2"          json:"eventId,omitempty"`
}

func (MessageModel) TableName() string {
	return "messages"
}
