package channel

import (
	"colossa-pm/chat"
	"colossa-pm/middlewares"
	"colossa-pm/workspace"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	chatRepo := chat.NewRepository(db)
	workspaceRepo := workspace.NewRepository(db)
	svc := NewService(repo, chatRepo, workspaceRepo)
	h := NewHandler(svc)

	ws := rg.Group("/workspaces/:workspaceId/channels")
	ws.Use(middlewares.Authenticate())
	{
		{
			ws.POST("", h.CreateChannel)
			ws.GET("", h.GetChannels)
			ws.GET("/me", h.GetMyChannels)
			ws.GET("/:channelId", h.GetChannel)
			ws.POST("/:channelId/join", h.JoinChannel)
			ws.POST("/:channelId/leave", h.LeaveChannel)
			ws.POST("/:channelId/members", h.AddMember)
			ws.POST("/:channelId/members/bulk", h.BulkAddMembers)
			ws.GET("/:channelId/members", h.GetMembers)
		}
	}
}
