package attachments

import (
	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Save(attachments []models.AttachmentModel) error
	FindByEntity(entityType string, entityID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.AttachmentModel], error)
	Delete(id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Save(items []models.AttachmentModel) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *repository) FindByEntity(entityType string, entityID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.AttachmentModel], error) {
	params.Normalize()

	var items []models.AttachmentModel
	var total int64

	query := r.db.Model(&models.AttachmentModel{}).Where("entity_type = ? AND entity_id = ?", entityType, entityID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Order("created_at ASC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&items).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(items, total, params), nil
}

func (r *repository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&models.AttachmentModel{}).Error
}
