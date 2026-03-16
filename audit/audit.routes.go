package audit

import (
	"colossa-pm/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, repo AuditRepository) {
	h := NewHandler(repo)

	routeGroup := rg.Group("/audit-logs")
	{
		routeGroup.Use(middlewares.Authenticate())
		routeGroup.GET("/:entityType/:entityId", h.GetByEntity)
	}
}
