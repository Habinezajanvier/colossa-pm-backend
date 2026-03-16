package workspace

import (
	"colossa-pm/email"
	"colossa-pm/messaging"
	"colossa-pm/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	mailer := email.NewMailer()
	messageRepo := messaging.NewRepository(db)
	svc := NewService(repo, mailer, messageRepo)
	h := NewHandler(svc)

	workspaces := rg.Group("/workspaces")
	workspaces.Use(middlewares.Authenticate())
	{
		workspaces.POST("", h.CreateWorkspace)
		workspaces.GET("", h.GetWorkspaces)
		workspaces.GET("/:workspaceId", h.GetWorkspace)
		workspaces.GET("/:workspaceId/members", h.GetMembers)
		workspaces.POST("/:workspaceId/invites", h.InviteMember)
		workspaces.POST("/invites/accept", h.AcceptInvite)
	}
}
