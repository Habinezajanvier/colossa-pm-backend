package chat

import (
	"errors"

	"context"
	"fmt"
	"mime/multipart"

	"colossa-pm/attachments"
	"colossa-pm/helpers"
	"colossa-pm/models"
	"colossa-pm/storage"
	"colossa-pm/workspace"

	"github.com/google/uuid"
)

var ErrNotWorkspaceMember = errors.New("both users must be members of the workspace")

// --- Input types ---

type StartConversationInput struct {
	WorkspaceID string `json:"workspaceId" binding:"required"`
	RecipientID string `json:"recipientId" binding:"required"`
}

type SendMessageInput struct {
	ConversationID string                  `json:"conversationId" binding:"required"`
	Body           *string                 `json:"body"`
	ParentID       *string                 `json:"parentId"`
	Files          []*multipart.FileHeader `json:"-"`
}

type AddReactionInput struct {
	Emoji string `json:"emoji" binding:"required"`
}

// --- Service ---

type Service interface {
	StartConversation(userID uuid.UUID, input StartConversationInput) (*models.ConversationModel, error)
	GetConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error)
	GetDMConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.DMConversation], error)
	GetParticipants(userID, conversationID uuid.UUID) ([]models.ConversationParticipantModel, error)
	GetMessages(userID uuid.UUID, conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error)
	GetReplies(userID uuid.UUID, conversationID, parentID uuid.UUID) ([]models.ChatMessageModel, error)
	SendMessage(senderID uuid.UUID, input SendMessageInput) (*models.ChatMessageModel, error)
	DeleteMessage(userID, messageID uuid.UUID) error
	AddReaction(userID, messageID uuid.UUID, input AddReactionInput) error
	RemoveReaction(userID, messageID uuid.UUID, emoji string) error
	VerifyParticipant(userID, conversationID uuid.UUID) error
}

type service struct {
	repo           Repository
	workspaceRepo  workspace.Repository
	attachmentRepo attachments.Repository
	hub            *Hub
	storage        storage.Client
}

func NewService(repo Repository, workspaceRepo workspace.Repository, attachmentRepo attachments.Repository, storageClient storage.Client) Service {
	return &service{
		repo:           repo,
		workspaceRepo:  workspaceRepo,
		attachmentRepo: attachmentRepo,
		hub:            GetHub(),
		storage:        storageClient,
	}
}

func (s *service) isParticipant(userID, conversationID uuid.UUID) error {
	_, err := s.repo.FindParticipant(conversationID, userID)
	return err
}

func (s *service) VerifyParticipant(userID, conversationID uuid.UUID) error {
	return s.isParticipant(userID, conversationID)
}

func (s *service) StartConversation(userID uuid.UUID, input StartConversationInput) (*models.ConversationModel, error) {
	workspaceID, err := uuid.Parse(input.WorkspaceID)
	if err != nil {
		return nil, errors.New("invalid workspaceId")
	}
	recipientID, err := uuid.Parse(input.RecipientID)
	if err != nil {
		return nil, errors.New("invalid recipientId")
	}

	// Verify both users are workspace members
	if _, err := s.workspaceRepo.FindMember(workspaceID, userID); err != nil {
		return nil, ErrNotWorkspaceMember
	}
	if _, err := s.workspaceRepo.FindMember(workspaceID, recipientID); err != nil {
		return nil, ErrNotWorkspaceMember
	}

	// Return existing DM if one already exists
	existing, err := s.repo.FindDMConversation(workspaceID, userID, recipientID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrConversationNotFound) {
		return nil, err
	}

	// Create new DM conversation
	conv := &models.ConversationModel{
		WorkspaceID: workspaceID,
		Type:        models.ConversationTypeDM,
	}
	if err := s.repo.CreateConversation(conv); err != nil {
		return nil, err
	}

	// Add both users as participants
	for _, uid := range []uuid.UUID{userID, recipientID} {
		if err := s.repo.AddParticipant(&models.ConversationParticipantModel{
			ConversationID: conv.ID,
			UserID:         uid,
			IsAdmin:        false,
		}); err != nil {
			return nil, err
		}
	}

	return conv, nil
}

func (s *service) GetDMConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.DMConversation], error) {
	return s.repo.FindDMConversationsByUser(userID, params)
}

func (s *service) GetConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error) {
	return s.repo.FindConversationsByUser(userID, params)
}

func (s *service) GetParticipants(userID, conversationID uuid.UUID) ([]models.ConversationParticipantModel, error) {
	if err := s.isParticipant(userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.FindParticipants(conversationID)
}

func (s *service) GetMessages(userID uuid.UUID, conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error) {
	if err := s.isParticipant(userID, conversationID); err != nil {
		return nil, err
	}

	result, err := s.repo.FindMessages(conversationID, params)
	if err != nil {
		return nil, err
	}

	s.hydrateFiles(result.Data)
	return result, nil
}

func (s *service) GetReplies(userID uuid.UUID, conversationID, parentID uuid.UUID) ([]models.ChatMessageModel, error) {
	if err := s.isParticipant(userID, conversationID); err != nil {
		return nil, err
	}

	replies, err := s.repo.FindReplies(parentID)
	if err != nil {
		return nil, err
	}

	s.hydrateFiles(replies)
	return replies, nil
}

func (s *service) SendMessage(senderID uuid.UUID, input SendMessageInput) (*models.ChatMessageModel, error) {
	conversationID, err := uuid.Parse(input.ConversationID)
	if err != nil {
		return nil, errors.New("invalid conversationId")
	}

	if err := s.isParticipant(senderID, conversationID); err != nil {
		return nil, err
	}

	if input.Body == nil && len(input.Files) == 0 {
		return nil, errors.New("message must have a body or at least one file")
	}

	msg := &models.ChatMessageModel{
		ConversationID: conversationID,
		SenderID:       senderID,
		Body:           input.Body,
	}

	if input.ParentID != nil {
		parentID, err := uuid.Parse(*input.ParentID)
		if err != nil {
			return nil, errors.New("invalid parentId")
		}
		msg.ParentID = &parentID
	}

	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}

	// Upload and save attachments
	if len(input.Files) > 0 {
		var items []models.AttachmentModel
		for _, fh := range input.Files {
			uploaded, err := s.storage.Upload(context.Background(), "chat", fh)
			if err != nil {
				return nil, fmt.Errorf("failed to upload file %s: %w", fh.Filename, err)
			}
			items = append(items, models.AttachmentModel{
				EntityType: "chat_message",
				EntityID:   msg.ID,
				URL:        uploaded.URL,
				Name:       uploaded.Name,
				Size:       uploaded.Size,
				MimeType:   uploaded.MimeType,
				FileType:   uploaded.FileType,
			})
		}
		if err := s.attachmentRepo.Save(items); err != nil {
			return nil, err
		}
	}

	// Reload with relations
	full, err := s.repo.FindMessageByID(msg.ID)
	if err != nil {
		return nil, err
	}

	if len(input.Files) > 0 {
		result, _ := s.attachmentRepo.FindByEntity("chat_message", msg.ID, helpers.PaginationParams{Page: 1, Limit: 100})
		if result != nil {
			full.Files = result.Data
		}
	}

	s.hub.Broadcast(conversationID, "message", full, nil)

	return full, nil
}

func (s *service) DeleteMessage(userID, messageID uuid.UUID) error {
	msg, err := s.repo.FindMessageByID(messageID)
	if err != nil {
		return err
	}
	if msg.SenderID != userID {
		return errors.New("you can only delete your own messages")
	}
	if err := s.repo.SoftDeleteMessage(messageID); err != nil {
		return err
	}

	s.hub.Broadcast(msg.ConversationID, "message_deleted", map[string]string{
		"messageId": messageID.String(),
	}, nil)

	return nil
}

func (s *service) AddReaction(userID, messageID uuid.UUID, input AddReactionInput) error {
	msg, err := s.repo.FindMessageByID(messageID)
	if err != nil {
		return err
	}

	if err := s.isParticipant(userID, msg.ConversationID); err != nil {
		return err
	}

	reaction := &models.MessageReactionModel{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     input.Emoji,
	}

	if err := s.repo.AddReaction(reaction); err != nil {
		return err
	}

	s.hub.Broadcast(msg.ConversationID, "reaction_added", reaction, nil)
	return nil
}

func (s *service) RemoveReaction(userID, messageID uuid.UUID, emoji string) error {
	msg, err := s.repo.FindMessageByID(messageID)
	if err != nil {
		return err
	}

	if err := s.repo.RemoveReaction(messageID, userID, emoji); err != nil {
		return err
	}

	s.hub.Broadcast(msg.ConversationID, "reaction_removed", map[string]string{
		"messageId": messageID.String(),
		"userId":    userID.String(),
		"emoji":     emoji,
	}, nil)

	return nil
}

// hydrateFiles loads attachments for a slice of messages in one query per message.
// Since Files is gorm:"-" we must load them manually after fetching messages.
func (s *service) hydrateFiles(messages []models.ChatMessageModel) {
	for i := range messages {
		result, _ := s.attachmentRepo.FindByEntity("chat_message", messages[i].ID, helpers.PaginationParams{Page: 1, Limit: 100})
		if result != nil {
			messages[i].Files = result.Data
		}
		// Also hydrate replies
		for j := range messages[i].Replies {
			replyResult, _ := s.attachmentRepo.FindByEntity("chat_message", messages[i].Replies[j].ID, helpers.PaginationParams{Page: 1, Limit: 100})
			if replyResult != nil {
				messages[i].Replies[j].Files = replyResult.Data
			}
		}
	}
}
