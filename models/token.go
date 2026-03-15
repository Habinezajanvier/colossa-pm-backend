package models

import (
	"time"

	"github.com/google/uuid"
)

type TokenType string

type VerificationTokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:(-)"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(64);not null;index"`
	Type      TokenType `gorm:"type:varchar(32);not null;index"`
	Used      bool      `gorm:"not null;default:false"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (VerificationTokenModel) TableName() string {
	return "verification_tokens"
}
