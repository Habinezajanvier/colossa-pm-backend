package channel

import (
	"errors"

	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrPrivateChannel  = errors.New("this channel is private — you must be invited to join")
)

type Repository interface {
	Create(channel *models.ChannelModel) error
	FindByID(id uuid.UUID) (*models.ChannelModel, error)
	FindByConversationID(conversationID uuid.UUID) (*models.ChannelModel, error)
	FindByWorkspace(workspaceID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error)
	FindMyChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(channel *models.ChannelModel) error {
	return r.db.Create(channel).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.ChannelModel, error) {
	var channel models.ChannelModel
	err := r.db.Where("id = ?", id).First(&channel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrChannelNotFound
	}
	return &channel, err
}

func (r *repository) FindByConversationID(conversationID uuid.UUID) (*models.ChannelModel, error) {
	var channel models.ChannelModel
	err := r.db.Where("conversation_id = ?", conversationID).First(&channel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrChannelNotFound
	}
	return &channel, err
}

func (r *repository) FindByWorkspace(workspaceID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error) {
	params.Normalize()

	var channels []models.ChannelModel
	var total int64

	query := r.db.Model(&models.ChannelModel{}).Where("workspace_id = ?", workspaceID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Order("created_at ASC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&channels).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(channels, total, params), nil
}

func (r *repository) FindMyChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error) {
	params.Normalize()

	var channels []models.ChannelModel
	var total int64

	// Public channels in the workspace OR private channels where user is a participant
	query := r.db.Model(&models.ChannelModel{}).
		Where("workspace_id = ?", workspaceID).
		Where(
			r.db.Where("is_private = FALSE").
				Or("id IN (?)",
					r.db.Model(&models.ChannelModel{}).
						Select("channels.id").
						Joins("JOIN conversation_participants cp ON cp.conversation_id = channels.conversation_id").
						Where("cp.user_id = ? AND channels.is_private = TRUE", userID),
				),
		)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Order("created_at ASC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&channels).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(channels, total, params), nil
}
