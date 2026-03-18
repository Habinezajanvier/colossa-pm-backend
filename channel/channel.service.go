package channel

import (
	"errors"

	"colossa-pm/chat"
	"colossa-pm/helpers"
	"colossa-pm/models"
	"colossa-pm/workspace"

	"github.com/google/uuid"
)

// --- Input types ---

type CreateChannelInput struct {
	Name        string  `json:"name"        binding:"required"`
	Description *string `json:"description"`
	IsPrivate   bool    `json:"isPrivate"`
}

type AddMemberInput struct {
	UserID string `json:"userId" binding:"required"`
}

type BulkAddMembersInput struct {
	UserIDs []string `json:"userIds" binding:"required,min=1"`
}

// --- Service ---

type Service interface {
	CreateChannel(workspaceID, userID uuid.UUID, input CreateChannelInput) (*models.ChannelModel, error)
	GetChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error)
	GetChannel(channelID, userID uuid.UUID) (*models.ChannelModel, error)
	JoinChannel(channelID, userID uuid.UUID) error
	AddMember(channelID, requesterID uuid.UUID, input AddMemberInput) error
	BulkAddMembers(channelID, requesterID uuid.UUID, input BulkAddMembersInput) (added int, errs []string, err error)
	GetMembers(channelID, userID uuid.UUID) ([]models.ConversationParticipantModel, error)
	GetMyChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error)
	LeaveChannel(channelID, userID uuid.UUID) error
	VerifyMember(userID, channelID uuid.UUID) error
}

type service struct {
	repo          Repository
	chatRepo      chat.Repository
	workspaceRepo workspace.Repository
}

func NewService(repo Repository, chatRepo chat.Repository, workspaceRepo workspace.Repository) Service {
	return &service{
		repo:          repo,
		chatRepo:      chatRepo,
		workspaceRepo: workspaceRepo,
	}
}

func (s *service) isMember(channelID, userID uuid.UUID) error {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return err
	}
	_, err = s.chatRepo.FindParticipant(ch.ConversationID, userID)
	return err
}

func (s *service) VerifyMember(userID, channelID uuid.UUID) error {
	return s.isMember(channelID, userID)
}

func (s *service) CreateChannel(workspaceID, userID uuid.UUID, input CreateChannelInput) (*models.ChannelModel, error) {
	if _, err := s.workspaceRepo.FindMember(workspaceID, userID); err != nil {
		return nil, errors.New("you must be a workspace member to create a channel")
	}

	// Create the shared conversation first
	conv := &models.ConversationModel{
		WorkspaceID: workspaceID,
		Type:        models.ConversationTypeChannel,
	}
	if err := s.chatRepo.CreateConversation(conv); err != nil {
		return nil, err
	}

	ch := &models.ChannelModel{
		WorkspaceID:    workspaceID,
		ConversationID: conv.ID,
		CreatedBy:      userID,
		Name:           input.Name,
		Description:    input.Description,
		IsPrivate:      input.IsPrivate,
	}

	if err := s.repo.Create(ch); err != nil {
		return nil, err
	}

	// Creator is automatically added as admin participant
	if err := s.chatRepo.AddParticipant(&models.ConversationParticipantModel{
		ConversationID: conv.ID,
		UserID:         userID,
		IsAdmin:        true,
	}); err != nil {
		return nil, err
	}

	return ch, nil
}

func (s *service) GetChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error) {
	if _, err := s.workspaceRepo.FindMember(workspaceID, userID); err != nil {
		return nil, errors.New("you must be a workspace member to view channels")
	}
	return s.repo.FindByWorkspace(workspaceID, params)
}

func (s *service) GetChannel(channelID, userID uuid.UUID) (*models.ChannelModel, error) {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return nil, err
	}

	if ch.IsPrivate {
		if err := s.isMember(channelID, userID); err != nil {
			return nil, ErrPrivateChannel
		}
	}

	return ch, nil
}

func (s *service) JoinChannel(channelID, userID uuid.UUID) error {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return err
	}

	if ch.IsPrivate {
		return ErrPrivateChannel
	}

	if _, err := s.workspaceRepo.FindMember(ch.WorkspaceID, userID); err != nil {
		return errors.New("you must be a workspace member to join this channel")
	}

	return s.chatRepo.AddParticipant(&models.ConversationParticipantModel{
		ConversationID: ch.ConversationID,
		UserID:         userID,
		IsAdmin:        false,
	})
}

func (s *service) AddMember(channelID, requesterID uuid.UUID, input AddMemberInput) error {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return err
	}

	// Requester must be a member
	requester, err := s.chatRepo.FindParticipant(ch.ConversationID, requesterID)
	if err != nil {
		return chat.ErrNotParticipant
	}

	// For private channels only admins can add members
	if ch.IsPrivate && !requester.IsAdmin {
		return errors.New("only channel admins can add members to a private channel")
	}

	newUserID, err := uuid.Parse(input.UserID)
	if err != nil {
		return errors.New("invalid userId")
	}

	if _, err := s.workspaceRepo.FindMember(ch.WorkspaceID, newUserID); err != nil {
		return errors.New("user must be a workspace member to join this channel")
	}

	return s.chatRepo.AddParticipant(&models.ConversationParticipantModel{
		ConversationID: ch.ConversationID,
		UserID:         newUserID,
		IsAdmin:        false,
	})
}

func (s *service) BulkAddMembers(channelID, requesterID uuid.UUID, input BulkAddMembersInput) (int, []string, error) {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return 0, nil, err
	}

	requester, err := s.chatRepo.FindParticipant(ch.ConversationID, requesterID)
	if err != nil {
		return 0, nil, chat.ErrNotParticipant
	}

	if ch.IsPrivate && !requester.IsAdmin {
		return 0, nil, errors.New("only channel admins can add members to a private channel")
	}

	added := 0
	var errs []string

	for _, rawID := range input.UserIDs {
		userID, err := uuid.Parse(rawID)
		if err != nil {
			errs = append(errs, rawID+": invalid userId")
			continue
		}

		if _, err := s.workspaceRepo.FindMember(ch.WorkspaceID, userID); err != nil {
			errs = append(errs, rawID+": user is not a workspace member")
			continue
		}

		if err := s.chatRepo.AddParticipant(&models.ConversationParticipantModel{
			ConversationID: ch.ConversationID,
			UserID:         userID,
			IsAdmin:        false,
		}); err != nil {
			if errors.Is(err, chat.ErrAlreadyParticipant) {
				errs = append(errs, rawID+": already a member")
			} else {
				errs = append(errs, rawID+": "+err.Error())
			}
			continue
		}
		added++
	}

	return added, errs, nil
}

func (s *service) GetMyChannels(workspaceID, userID uuid.UUID, params helpers.PaginationParams) (*helpers.PaginatedResult[models.ChannelModel], error) {
	if _, err := s.workspaceRepo.FindMember(workspaceID, userID); err != nil {
		return nil, errors.New("you must be a workspace member to view channels")
	}
	return s.repo.FindMyChannels(workspaceID, userID, params)
}

func (s *service) GetMembers(channelID, userID uuid.UUID) ([]models.ConversationParticipantModel, error) {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return nil, err
	}

	// Private channel — only members can view member list
	if ch.IsPrivate {
		if err := s.isMember(channelID, userID); err != nil {
			return nil, ErrPrivateChannel
		}
	}

	return s.chatRepo.FindParticipants(ch.ConversationID)
}

func (s *service) LeaveChannel(channelID, userID uuid.UUID) error {
	ch, err := s.repo.FindByID(channelID)
	if err != nil {
		return err
	}
	return s.chatRepo.RemoveParticipant(ch.ConversationID, userID)
}
