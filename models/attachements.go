package models

import (
	"colossa-pm/storage"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AttachmentModel struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey;default:(-)"        json:"id"`
	EntityType string           `gorm:"type:varchar(100);not null;index:idx_entity" json:"entityType"`
	EntityID   uuid.UUID        `gorm:"type:uuid;not null;index:idx_entity"     json:"entityId"`
	URL        string           `gorm:"type:text;not null"                      json:"url"`
	Name       string           `gorm:"type:varchar(255);not null"              json:"name"`
	Size       int64            `gorm:"not null"                                json:"size"`
	MimeType   string           `gorm:"type:varchar(100);not null"              json:"mimeType"`
	FileType   storage.FileType `gorm:"type:varchar(20);not null"               json:"fileType"`
	CreatedAt  time.Time        `                                               json:"createdAt"`
}

func (a *AttachmentModel) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (AttachmentModel) TableName() string {
	return "attachments"
}
