package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RequestMeta struct {
	IPAddress string `json:"ipAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	Method    string `json:"method,omitempty"`
	URL       string `json:"url,omitempty"`
}

// AuditLog represents a single recorded activity on the platform
type AuditLogModel struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey;default:(-)"  json:"id"`
	UserID       *uuid.UUID   `gorm:"type:uuid;index"                   json:"userId,omitempty"`
	UserFullName string       `gorm:"type:varchar(255)"                 json:"fullName,omitempty"`
	Action       string       `gorm:"type:varchar(100);not null;index"  json:"action"`
	EntityType   *string      `gorm:"type:varchar(100);index"           json:"entityType,omitempty"`
	EntityID     *uuid.UUID   `gorm:"type:uuid"                         json:"entityId,omitempty"`
	OldValues    *JSON        `gorm:"type:jsonb"                        json:"oldValues,omitempty"`
	NewValues    *JSON        `gorm:"type:jsonb"                        json:"newValues,omitempty"`
	Request      *RequestMeta `gorm:"type:jsonb;serializer:json"        json:"request,omitempty"`
	CreatedAt    time.Time    `                                         json:"createdAt"`
}

type JSON map[string]interface{}

func (j JSON) Value() ([]byte, error) {
	return json.Marshal(j)
}

func (u *AuditLogModel) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (AuditLogModel) TableName() string {
	return "audit_logs"
}
