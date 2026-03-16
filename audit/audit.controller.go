package audit

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles audit log HTTP endpoints
type Handler struct {
	repo AuditRepository
}

func NewHandler(repo AuditRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetByEntity(c *gin.Context) {
	entityType := c.Param("entityType")
	entityIDStr := c.Param("entityId")

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entityId"})
		return
	}

	logs, err := h.repo.FindByEntity(entityType, entityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}
