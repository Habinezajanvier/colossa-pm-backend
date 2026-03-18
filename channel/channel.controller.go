package channel

import (
	"errors"
	"net/http"

	"colossa-pm/audit"
	"colossa-pm/chat"
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

func (h *Handler) CreateChannel(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	var input CreateChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.CreateChannel(workspaceID, userID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.SetAction(c, "channel.created")
	audit.SetEntity(c, "channel", ch.ID)
	c.JSON(http.StatusCreated, ch)
}

func (h *Handler) GetChannels(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetChannels(workspaceID, userID, params)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	audit.SetAction(c, "channel.list")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetChannel(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	ch, err := h.svc.GetChannel(channelID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrChannelNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrPrivateChannel):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch channel"})
		}
		return
	}

	audit.SetAction(c, "channel.viewed")
	audit.SetEntity(c, "channel", ch.ID)

	// Return channel with its conversationId so client can use chat endpoints
	c.JSON(http.StatusOK, ch)
}

func (h *Handler) JoinChannel(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	if err := h.svc.JoinChannel(channelID, userID); err != nil {
		switch {
		case errors.Is(err, ErrPrivateChannel):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, chat.ErrAlreadyParticipant):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	audit.SetAction(c, "channel.joined")
	audit.SetEntity(c, "channel", channelID)
	c.JSON(http.StatusOK, gin.H{"message": "joined channel successfully"})
}

func (h *Handler) AddMember(c *gin.Context) {
	requesterID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	var input AddMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.AddMember(channelID, requesterID, input); err != nil {
		switch {
		case errors.Is(err, chat.ErrNotParticipant):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case errors.Is(err, chat.ErrAlreadyParticipant):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	audit.SetAction(c, "channel.member_added")
	audit.SetEntity(c, "channel", channelID)
	c.JSON(http.StatusOK, gin.H{"message": "member added successfully"})
}

func (h *Handler) BulkAddMembers(c *gin.Context) {
	requesterID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	var input BulkAddMembersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	added, errs, err := h.svc.BulkAddMembers(channelID, requesterID, input)
	if err != nil {
		switch {
		case errors.Is(err, chat.ErrNotParticipant):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	audit.SetAction(c, "channel.members_bulk_added")
	audit.SetEntity(c, "channel", channelID)
	c.JSON(http.StatusOK, gin.H{
		"added":  added,
		"failed": errs,
	})
}

func (h *Handler) GetMyChannels(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	workspaceID, err := uuid.Parse(c.Param("workspaceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspaceId"})
		return
	}

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params.Normalize()

	result, err := h.svc.GetMyChannels(workspaceID, userID, params)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	audit.SetAction(c, "channel.my_channels_listed")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetMembers(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	members, err := h.svc.GetMembers(channelID, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrChannelNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, ErrPrivateChannel):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch members"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": members})
}

func (h *Handler) LeaveChannel(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channelId"})
		return
	}

	if err := h.svc.LeaveChannel(channelID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	audit.SetAction(c, "channel.left")
	audit.SetEntity(c, "channel", channelID)
	c.JSON(http.StatusOK, gin.H{"message": "left channel successfully"})
}
