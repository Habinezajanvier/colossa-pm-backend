package messaging

import (
	"colossa-pm/models"

	"gorm.io/gorm"
)

const (
	MessageTypeEmailVerification models.MessageType = "email_verification"
	MessageTypeChangePassword    models.MessageType = "change_password"
)

const (
	MessageStatusSent   models.MessageStatus = "sent"
	MessageStatusFailed models.MessageStatus = "failed"
)

type Repository interface {
	Save(message *models.MessageModel) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Save(message *models.MessageModel) error {
	return r.db.Create(message).Error
}
