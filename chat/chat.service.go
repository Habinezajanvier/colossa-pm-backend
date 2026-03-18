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
	Files          []*multipart.FileHeader `json:"-"` // set from multipart form
}

type AddReactionInput struct {
	Emoji string `json:"emoji" binding:"required"`
}

// --- Service ---

type Service interface {
	StartConversation(userID uuid.UUID, input StartConversationInput) (*models.ConversationModel, error)
	GetConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error)
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

func (s *service) StartConversation(userID uuid.UUID, input StartConversationInput) (*models.ConversationModel, error) {
	workspaceID, err := uuid.Parse(input.WorkspaceID)
	if err != nil {
		return nil, errors.New("invalid workspaceId")
	}
	recipientID, err := uuid.Parse(input.RecipientID)
	if err != nil {
		return nil, errors.New("invalid recipientId")
	}

	// Verify both users are members of the workspace
	if _, err := s.workspaceRepo.FindMember(workspaceID, userID); err != nil {
		return nil, ErrNotWorkspaceMember
	}
	if _, err := s.workspaceRepo.FindMember(workspaceID, recipientID); err != nil {
		return nil, ErrNotWorkspaceMember
	}

	return s.repo.FindOrCreateConversation(workspaceID, userID, recipientID)
}

func (s *service) GetConversations(userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ConversationModel], error) {
	return s.repo.FindConversationsByUser(userID, params)
}

func (s *service) isParticipant(userID, conversationID uuid.UUID) error {
	conv, err := s.repo.FindConversation(conversationID)
	if err != nil {
		return err
	}
	if conv.MemberOne != userID && conv.MemberTwo != userID {
		return ErrNotParticipant
	}
	return nil
}

func (s *service) GetMessages(userID uuid.UUID, conversationID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChatMessageModel], error) {
	if err := s.isParticipant(userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.FindMessages(conversationID, params)
}

func (s *service) GetReplies(userID uuid.UUID, conversationID, parentID uuid.UUID) ([]models.ChatMessageModel, error) {
	if err := s.isParticipant(userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.FindReplies(parentID)
}

// VerifyParticipant is exported for use in the WebSocket handler
func (s *service) VerifyParticipant(userID, conversationID uuid.UUID) error {
	return s.isParticipant(userID, conversationID)
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

	// Reload with relations for the broadcast payload
	full, err := s.repo.FindMessageByID(msg.ID)
	if err != nil {
		return nil, err
	}

	// Attach uploaded files to response
	if len(input.Files) > 0 {
		result, _ := s.attachmentRepo.FindByEntity("chat_message", msg.ID, helpers.PaginationParams{Page: 1, Limit: 100})
		if result != nil {
			full.Files = result.Data
		}
	}

	// Broadcast to all WebSocket clients in the conversation
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
