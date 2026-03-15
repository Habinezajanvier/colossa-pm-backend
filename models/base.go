package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:(-)" json:"id"`
	CreatedBy      uuid.UUID      ` json:"createdBy"`
	CreatedByNames string         ` json:"createdByFullName"`
	UpdatedBy      uuid.UUID      ` json:"updatedBy"`
	UpdatedByNames string         ` json:"updatedByFullName"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedAt      time.Time      `                                         json:"createdAt"`
	UpdatedAt      time.Time      `                                         json:"updatedAt"`
}

// BeforeCreate generates a UUID before inserting a new record
func (model *Base) BeforeCreate(tx *gorm.DB) error {
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	return nil
}
