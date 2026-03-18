package chat

import (
	"errors"

	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotParticipant       = errors.New("you are not a participant of this conversation")
	ErrAlreadyReacted       = errors.New("you have already reacted with this emoji")
)

type Repository interface {
	// Conversations
	FindOrCreateConversation(workspaceID, memberOne, memberTwo uuid.UUID) (*models.ConversationModel, error)
	FindConversation(id uuid.UUID) (*models.ConversationModel, error)
	FindConversationsByUser(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error)

	// Messages
	CreateMessage(msg *models.ChatMessageModel) error
	FindMessages(conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error)
	FindReplies(parentID uuid.UUID) ([]models.ChatMessageModel, error)
	FindMessageByID(id uuid.UUID) (*models.ChatMessageModel, error)
	SoftDeleteMessage(id uuid.UUID) error

	// Reactions
	AddReaction(reaction *models.MessageReactionModel) error
	RemoveReaction(messageID, userID uuid.UUID, emoji string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Conversations ---

// FindOrCreateConversation ensures member_one < member_two for the unique constraint
func (r *repository) FindOrCreateConversation(workspaceID, memberOne, memberTwo uuid.UUID) (*models.ConversationModel, error) {
	// Enforce consistent ordering
	if memberOne.String() > memberTwo.String() {
		memberOne, memberTwo = memberTwo, memberOne
	}

	var conv models.ConversationModel
	err := r.db.Where("workspace_id = ? AND member_one = ? AND member_two = ?", workspaceID, memberOne, memberTwo).
		First(&conv).Error

	if err == nil {
		return &conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	conv = models.ConversationModel{
		WorkspaceID: workspaceID,
		MemberOne:   memberOne,
		MemberTwo:   memberTwo,
	}
	if err := r.db.Create(&conv).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *repository) FindConversation(id uuid.UUID) (*models.ConversationModel, error) {
	var conv models.ConversationModel
	err := r.db.Where("id = ?", id).First(&conv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConversationNotFound
	}
	return &conv, err
}

func (r *repository) FindConversationsByUser(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error) {
	params.Normalize()

	var conversations []models.ConversationModel
	var total int64

	query := r.db.Model(&models.ConversationModel{}).Where("member_one = ? OR member_two = ?", userID, userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(conversations, total, params), nil
}

// --- Messages ---

func (r *repository) CreateMessage(msg *models.ChatMessageModel) error {
	return r.db.Create(msg).Error
}

func (r *repository) FindMessages(conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error) {
	params.Normalize()

	var messages []models.ChatMessageModel
	var total int64

	// Only top-level messages (no parent) — replies fetched separately
	query := r.db.Model(&models.ChatMessageModel{}).
		Where("conversation_id = ? AND parent_id IS NULL AND deleted_at IS NULL", conversationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Preload("Sender").
		Preload("Reactions.User").
		Preload("Replies.Sender").
		Order("created_at ASC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(messages, total, params), nil
}

func (r *repository) FindReplies(parentID uuid.UUID) ([]models.ChatMessageModel, error) {
	var replies []models.ChatMessageModel
	err := r.db.
		Preload("Sender").
		Preload("Reactions.User").
		Where("parent_id = ? AND deleted_at IS NULL", parentID).
		Order("created_at ASC").
		Find(&replies).Error
	return replies, err
}

func (r *repository) FindMessageByID(id uuid.UUID) (*models.ChatMessageModel, error) {
	var msg models.ChatMessageModel
	err := r.db.
		Preload("Sender").
		Preload("Reactions.User").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&msg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMessageNotFound
	}
	return &msg, err
}

func (r *repository) SoftDeleteMessage(id uuid.UUID) error {
	result := r.db.Model(&models.ChatMessageModel{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()"))
	if result.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return result.Error
}

// --- Reactions ---

func (r *repository) AddReaction(reaction *models.MessageReactionModel) error {
	result := r.db.Create(reaction)
	if result.Error != nil {
		if result.Error.Error() != "" {
			return ErrAlreadyReacted
		}
		return result.Error
	}
	return nil
}

func (r *repository) RemoveReaction(messageID, userID uuid.UUID, emoji string) error {
	return r.db.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&models.MessageReactionModel{}).Error
}
