package chat

import (
	"errors"
	"time"

	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrNotParticipant       = errors.New("you are not a participant of this conversation")
	ErrAlreadyParticipant   = errors.New("user is already a participant")
	ErrAlreadyReacted       = errors.New("you have already reacted with this emoji")
)

type Repository interface {
	// Conversations
	CreateConversation(conv *models.ConversationModel) error
	FindConversation(id uuid.UUID) (*models.ConversationModel, error)
	FindConversationsByUser(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error)
	FindDMConversationsByUser(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.DMConversation], error)
	FindDMConversation(workspaceID, userOne, userTwo uuid.UUID) (*models.ConversationModel, error)

	// Participants
	AddParticipant(p *models.ConversationParticipantModel) error
	FindParticipant(conversationID, userID uuid.UUID) (*models.ConversationParticipantModel, error)
	FindParticipants(conversationID uuid.UUID) ([]models.ConversationParticipantModel, error)
	RemoveParticipant(conversationID, userID uuid.UUID) error

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

func (r *repository) CreateConversation(conv *models.ConversationModel) error {
	return r.db.Create(conv).Error
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

	query := r.db.Model(&models.ConversationModel{}).
		Joins("JOIN conversation_participants cp ON cp.conversation_id = conversations.id").
		Where("cp.user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Preload("Participants.User").
		Order("conversations.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(conversations, total, params), nil
}

func (r *repository) FindDMConversationsByUser(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.DMConversation], error) {
	params.Normalize()

	var conversations []models.ConversationModel
	var total int64

	query := r.db.Model(&models.ConversationModel{}).
		Joins("JOIN conversation_participants cp ON cp.conversation_id = conversations.id").
		Where("cp.user_id = ? AND conversations.type = ?", userID, models.ConversationTypeDM)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Order("conversations.created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&conversations).Error; err != nil {
		return nil, err
	}

	// For each DM, find all other participants
	result := make([]models.DMConversation, 0, len(conversations))
	for _, conv := range conversations {
		dm := models.DMConversation{
			ConversationModel: conv,
			OtherParticipants: []models.UsersModel{},
		}

		var otherParticipants []models.ConversationParticipantModel
		err := r.db.Preload("User").
			Where("conversation_id = ? AND user_id != ?", conv.ID, userID).
			Find(&otherParticipants).Error
		if err == nil {
			for _, p := range otherParticipants {
				if p.User != nil {
					dm.OtherParticipants = append(dm.OtherParticipants, *p.User)
				}
			}
		}

		result = append(result, dm)
	}

	return helpers.NewPaginatedResult(result, total, params), nil
}

// FindDMConversation finds an existing DM between two users in a workspace
func (r *repository) FindDMConversation(workspaceID, userOne, userTwo uuid.UUID) (*models.ConversationModel, error) {
	var conv models.ConversationModel
	err := r.db.
		Joins("JOIN conversation_participants cp1 ON cp1.conversation_id = conversations.id AND cp1.user_id = ?", userOne).
		Joins("JOIN conversation_participants cp2 ON cp2.conversation_id = conversations.id AND cp2.user_id = ?", userTwo).
		Where("conversations.workspace_id = ? AND conversations.type = ?", workspaceID, models.ConversationTypeDM).
		First(&conv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConversationNotFound
	}
	return &conv, err
}

// --- Participants ---

func (r *repository) AddParticipant(p *models.ConversationParticipantModel) error {
	var existing models.ConversationParticipantModel
	err := r.db.Where("conversation_id = ? AND user_id = ?", p.ConversationID, p.UserID).
		First(&existing).Error
	if err == nil {
		return ErrAlreadyParticipant
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	p.JoinedAt = time.Now()
	return r.db.Create(p).Error
}

func (r *repository) FindParticipant(conversationID, userID uuid.UUID) (*models.ConversationParticipantModel, error) {
	var p models.ConversationParticipantModel
	err := r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotParticipant
	}
	return &p, err
}

func (r *repository) FindParticipants(conversationID uuid.UUID) ([]models.ConversationParticipantModel, error) {
	var participants []models.ConversationParticipantModel
	err := r.db.Preload("User").
		Where("conversation_id = ?", conversationID).
		Order("joined_at ASC").
		Find(&participants).Error
	return participants, err
}

func (r *repository) RemoveParticipant(conversationID, userID uuid.UUID) error {
	return r.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Delete(&models.ConversationParticipantModel{}).Error
}

// --- Messages ---

func (r *repository) CreateMessage(msg *models.ChatMessageModel) error {
	return r.db.Create(msg).Error
}

func (r *repository) FindMessages(conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error) {
	params.Normalize()

	var messages []models.ChatMessageModel
	var total int64

	query := r.db.Model(&models.ChatMessageModel{}).
		Where("conversation_id = ? AND parent_id IS NULL AND deleted_at IS NULL", conversationID)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Preload("Sender").
		Preload("Reactions.User").
		Preload("Replies.Sender").
		Preload("Replies.Reactions.User").
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
	if err := r.db.Create(reaction).Error; err != nil {
		return ErrAlreadyReacted
	}
	return nil
}

func (r *repository) RemoveReaction(messageID, userID uuid.UUID, emoji string) error {
	return r.db.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&models.MessageReactionModel{}).Error
}
