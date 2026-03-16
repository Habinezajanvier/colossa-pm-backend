package workspace

import (
	"errors"
	"fmt"
	"net/http"

	"colossa-pm/audit"
	"colossa-pm/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateWorkspace(c *gin.Context) {
	ownerID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input CreateWorkspaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	workspace, err := h.svc.CreateWorkspace(ownerID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workspace"})
		return
	}

	audit.SetAction(c, "workspace.created")
	audit.SetEntity(c, "workspace", workspace.ID)
	c.JSON(http.StatusCreated, workspace)
}

func (h *Handler) GetWorkspace(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	workspace, err := h.svc.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, ErrWorkspaceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch workspace"})
		return
	}

	audit.SetAction(c, "workspace.viewed")
	audit.SetEntity(c, "workspace", workspace.ID)
	c.JSON(http.StatusOK, workspace)
}

func (h *Handler) GetWorkspaces(c *gin.Context) {
	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetWorkspaces(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch workspaces"})
		return
	}

	audit.SetAction(c, "workspace.list")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetMembers(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	members, err := h.svc.GetMembers(workspaceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch members"})
		return
	}

	audit.SetAction(c, "workspace.members_listed")
	audit.SetEntity(c, "workspace", workspaceID)
	c.JSON(http.StatusOK, gin.H{"data": members})
}

func (h *Handler) InviteMember(c *gin.Context) {
	inviterID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	var input InviteMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.InviteMember(workspaceID, inviterID, input); err != nil {
		enviteError := fmt.Sprintf("===error-sending===> %s", err)
		fmt.Println(enviteError)
		switch {
		case errors.Is(err, ErrNotMember):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrNotAdmin):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, ErrWorkspaceNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send invite"})
		}
		return
	}

	audit.SetAction(c, "workspace.member_invited")
	audit.SetEntity(c, "workspace", workspaceID)
	c.JSON(http.StatusOK, gin.H{"message": "invite sent successfully"})
}

func (h *Handler) AcceptInvite(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input AcceptInviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.svc.AcceptInvite(userID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, ErrInviteExpired):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, ErrAlreadyMember):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept invite"})
		}
		return
	}

	audit.SetAction(c, "workspace.member_joined")
	audit.SetEntity(c, "workspace", member.WorkspaceID)
	c.JSON(http.StatusOK, member)
}
