package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"colossa-pm/helpers"
	"colossa-pm/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrAlreadyMember     = errors.New("user is already a member of this workspace")
	ErrNotMember         = errors.New("user is not a member of this workspace")
	ErrInviteNotFound    = errors.New("invite not found or already used")
	ErrInviteExpired     = errors.New("invite has expired")
	ErrNotAdmin          = errors.New("only admins can perform this action")
)

type Repository interface {
	// Workspace
	Create(workspace *models.WorkspaceModel) error
	FindByID(id uuid.UUID) (*models.WorkspaceModel, error)
	FindAll(params helpers.PaginationParams) (*helpers.PaginatedResult[models.WorkspaceModel], error)

	// Members
	AddMember(member *models.WorkspaceMemberModel) error
	FindMember(workspaceID, userID uuid.UUID) (*models.WorkspaceMemberModel, error)
	FindMembers(workspaceID uuid.UUID) ([]models.WorkspaceMemberModel, error)

	// Invites
	CreateInvite(invite *models.WorkspaceInviteModel) error
	FindInviteByToken(token string) (*models.WorkspaceInviteModel, error)
	FindInviteByEmail(workspaceID uuid.UUID, email string) (*models.WorkspaceInviteModel, error)
	MarkInviteUsed(id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// --- Workspace ---

func (r *repository) Create(workspace *models.WorkspaceModel) error {
	return r.db.Create(workspace).Error
}

func (r *repository) FindByID(id uuid.UUID) (*models.WorkspaceModel, error) {
	var workspace models.WorkspaceModel
	err := r.db.Where("id = ?", id).First(&workspace).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkspaceNotFound
	}
	return &workspace, err
}

func (r *repository) FindAll(params helpers.PaginationParams) (*helpers.PaginatedResult[models.WorkspaceModel], error) {
	params.Normalize()

	var workspaces []models.WorkspaceModel
	var total int64

	if err := r.db.Model(&models.WorkspaceModel{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := r.db.Order("created_at DESC").
		Limit(params.Limit).
		Offset(params.Offset()).
		Find(&workspaces).Error; err != nil {
		return nil, err
	}

	return helpers.NewPaginatedResult(workspaces, total, params), nil
}

// --- Members ---

func (r *repository) AddMember(member *models.WorkspaceMemberModel) error {
	// Check if already a member
	var existing models.WorkspaceMemberModel
	err := r.db.Where("workspace_id = ? AND user_id = ?", member.WorkspaceID, member.UserID).
		First(&existing).Error
	if err == nil {
		return ErrAlreadyMember
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	member.JoinedAt = time.Now()
	return r.db.Create(member).Error
}

func (r *repository) FindMember(workspaceID, userID uuid.UUID) (*models.WorkspaceMemberModel, error) {
	var member models.WorkspaceMemberModel
	err := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotMember
	}
	return &member, err
}

func (r *repository) FindMembers(workspaceID uuid.UUID) ([]models.WorkspaceMemberModel, error) {
	var members []models.WorkspaceMemberModel
	err := r.db.Preload("User").
		Where("workspace_id = ?", workspaceID).
		Order("joined_at ASC").
		Find(&members).Error
	return members, err
}

// --- Invites ---

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (r *repository) CreateInvite(invite *models.WorkspaceInviteModel) error {
	// Invalidate any previous unused invite for same email + workspace
	r.db.Model(&models.WorkspaceInviteModel{}).
		Where("workspace_id = ? AND email = ? AND used = FALSE", invite.WorkspaceID, invite.Email).
		Update("used", true)

	invite.Token = hashToken(invite.Token)
	return r.db.Create(invite).Error
}

func (r *repository) FindInviteByToken(token string) (*models.WorkspaceInviteModel, error) {
	var invite models.WorkspaceInviteModel
	err := r.db.Where("token = ? AND used = FALSE AND expires_at > ?", hashToken(token), time.Now()).
		First(&invite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInviteNotFound
	}
	return &invite, err
}

func (r *repository) FindInviteByEmail(workspaceID uuid.UUID, email string) (*models.WorkspaceInviteModel, error) {
	var invite models.WorkspaceInviteModel
	err := r.db.Where("workspace_id = ? AND email = ? AND used = FALSE AND expires_at > ?",
		workspaceID, email, time.Now()).
		First(&invite).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInviteNotFound
	}
	return &invite, err
}

func (r *repository) MarkInviteUsed(id uuid.UUID) error {
	return r.db.Model(&models.WorkspaceInviteModel{}).Where("id = ?", id).Update("used", true).Error
}
