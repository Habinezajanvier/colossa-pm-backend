package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"colossa-pm/email"
	"colossa-pm/helpers"
	applogger "colossa-pm/logger"
	"colossa-pm/messaging"
	"colossa-pm/models"

	"github.com/google/uuid"
)

// --- Input types ---

type CreateWorkspaceInput struct {
	Name        string  `json:"name"        binding:"required"`
	Description *string `json:"description"`
}

type InviteMemberInput struct {
	Email string      `json:"email" binding:"required,email"`
	Role  models.Role `json:"role"  binding:"required"`
}

type AcceptInviteInput struct {
	Token string `json:"token" binding:"required"`
}

// --- Service ---

type Service interface {
	CreateWorkspace(ownerID uuid.UUID, input CreateWorkspaceInput) (*models.WorkspaceModel, error)
	GetWorkspace(id uuid.UUID) (*models.WorkspaceModel, error)
	GetWorkspaces(params helpers.PaginationParams) (*helpers.PaginatedResult[models.WorkspaceModel], error)
	GetMembers(workspaceID uuid.UUID) ([]models.WorkspaceMemberModel, error)
	InviteMember(workspaceID, inviterID uuid.UUID, input InviteMemberInput) error
	AcceptInvite(userID uuid.UUID, input AcceptInviteInput) (*models.WorkspaceMemberModel, error)
}

type service struct {
	repo        Repository
	mailer      email.Mailer
	messageRepo messaging.Repository
}

func NewService(repo Repository, mailer email.Mailer, messageRepo messaging.Repository) Service {
	return &service{repo: repo, mailer: mailer, messageRepo: messageRepo}
}

func (s *service) CreateWorkspace(ownerID uuid.UUID, input CreateWorkspaceInput) (*models.WorkspaceModel, error) {
	workspace := &models.WorkspaceModel{
		Name:        input.Name,
		Description: input.Description,
		OwnerID:     ownerID,
	}

	if err := s.repo.Create(workspace); err != nil {
		return nil, err
	}

	// Owner is automatically added as admin
	if err := s.repo.AddMember(&models.WorkspaceMemberModel{
		WorkspaceID: workspace.ID,
		UserID:      ownerID,
		Role:        models.RoleAdmin,
	}); err != nil {
		return nil, err
	}

	return workspace, nil
}

func (s *service) GetWorkspace(id uuid.UUID) (*models.WorkspaceModel, error) {
	return s.repo.FindByID(id)
}

func (s *service) GetWorkspaces(params helpers.PaginationParams) (*helpers.PaginatedResult[models.WorkspaceModel], error) {
	return s.repo.FindAll(params)
}

func (s *service) GetMembers(workspaceID uuid.UUID) ([]models.WorkspaceMemberModel, error) {
	return s.repo.FindMembers(workspaceID)
}

func (s *service) InviteMember(workspaceID, inviterID uuid.UUID, input InviteMemberInput) error {
	// Verify inviter is an admin
	member, err := s.repo.FindMember(workspaceID, inviterID)
	if err != nil {
		return err
	}
	if member.Role != models.RoleAdmin {
		return ErrNotAdmin
	}

	workspace, err := s.repo.FindByID(workspaceID)
	if err != nil {
		return err
	}

	// Generate a secure random token
	rawToken, err := generateInviteToken()
	if err != nil {
		return err
	}

	invite := &models.WorkspaceInviteModel{
		WorkspaceID: workspaceID,
		InvitedBy:   inviterID,
		Email:       input.Email,
		Role:        input.Role,
		Token:       rawToken,
		ExpiresAt:   time.Now().Add(48 * time.Hour),
	}

	if err := s.repo.CreateInvite(invite); err != nil {
		return err
	}

	// Send invite email asynchronously
	go func() {
		eventType := "workspace.invite"
		msg := &models.MessageModel{
			UserID:    inviterID,
			Recipient: input.Email,
			Type:      models.MessageType(eventType),
			Status:    messaging.MessageStatusSent,
			EventType: &eventType,
			EventID:   &invite.ID,
		}

		subject := fmt.Sprintf("You've been invited to join %s", workspace.Name)
		body := email.BuildInviteEmailBody(workspace.Name, rawToken)

		if err := s.mailer.SendRaw(input.Email, subject, body); err != nil {
			errStr := err.Error()
			msg.Status = messaging.MessageStatusFailed
			msg.Error = &errStr
			applogger.Instance().ErrorMsg("failed to send workspace invite email: " + errStr)
		} else {
			applogger.Instance().Log("workspace invite email sent to " + input.Email)
		}

		if err := s.messageRepo.Save(msg); err != nil {
			applogger.Instance().ErrorMsg("failed to save message log: " + err.Error())
		}
	}()

	return nil
}

func (s *service) AcceptInvite(userID uuid.UUID, input AcceptInviteInput) (*models.WorkspaceMemberModel, error) {
	invite, err := s.repo.FindInviteByToken(input.Token)
	if err != nil {
		return nil, err
	}

	if err := s.repo.MarkInviteUsed(invite.ID); err != nil {
		return nil, err
	}

	member := &models.WorkspaceMemberModel{
		WorkspaceID: invite.WorkspaceID,
		UserID:      userID,
		Role:        invite.Role,
	}

	if err := s.repo.AddMember(member); err != nil {
		return nil, err
	}

	return member, nil
}

// --- Helpers ---

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate invite token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
