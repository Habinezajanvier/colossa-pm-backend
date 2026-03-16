package users

import (
	"net/http"

	"colossa-pm/audit"
	"colossa-pm/helpers"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetUsers(c *gin.Context) {
	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.GetUsers(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}

	audit.SetAction(c, "user.list")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := helpers.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.svc.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile"})
		return
	}

	audit.SetAction(c, "user.profile_viewed")
	audit.SetEntity(c, "user", profile.ID)
	c.JSON(http.StatusOK, profile)
}
