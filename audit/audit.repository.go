package audit

import (
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Save(log *models.AuditLogModel) error
	FindByEntity(entityType string, entityID uuid.UUID) ([]models.AuditLogModel, error)
}

type repository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &repository{db: db}
}

func (r *repository) Save(log *models.AuditLogModel) error {
	return r.db.Create(log).Error
}

func (r *repository) FindByEntity(entityType string, entityID uuid.UUID) ([]models.AuditLogModel, error) {
	var logs []models.AuditLogModel
	err := r.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}
