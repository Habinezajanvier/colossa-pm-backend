package audit

import (
	"colossa-pm/helpers"
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

	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entityId"})
		return
	}
	params.Normalize()

	result, err := h.repo.FindByEntity(entityType, entityID, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetAll(c *gin.Context) {
	var params helpers.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params.Normalize()
	result, err := h.repo.FindAll(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}

	c.JSON(http.StatusOK, result)
}
