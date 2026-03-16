package audit

import (
	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditRepository interface {
	Save(log *models.AuditLogModel) error
	FindByEntity(entityType string, entityID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.AuditLogModel], error)
	FindAll(params helpers.PaginationParams) (*helpers.PaginatedResult[models.AuditLogModel], error)
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

func (r *repository) FindByEntity(entityType string, entityID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.AuditLogModel], error) {

	var logs []models.AuditLogModel
	var total int64

	if err := r.db.Model(&models.AuditLogModel{}).Where("entity_type = ? AND entity_id = ?", entityType, entityID).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.AuditLogModel{}).Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return helpers.NewPaginatedResult(logs, total, params), nil
}

func (r *repository) FindAll(params helpers.PaginationParams) (*helpers.PaginatedResult[models.AuditLogModel], error) {

	var logs []models.AuditLogModel
	var total int64

	if err := r.db.Model(&models.AuditLogModel{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := r.db.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(logs, total, params), nil
}
