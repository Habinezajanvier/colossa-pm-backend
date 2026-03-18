package chat

import (
	"context"
	"log"

	"colossa-pm/attachments"
	"colossa-pm/middlewares"
	"colossa-pm/storage"
	"colossa-pm/workspace"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	storageClient, err := storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("failed to initialize storage client: %v", err)
	}

	repo := NewRepository(db)
	workspaceRepo := workspace.NewRepository(db)
	attachmentRepo := attachments.NewRepository(db)
	svc := NewService(repo, workspaceRepo, attachmentRepo, storageClient)
	h := NewHandler(svc)

	chat := rg.Group("/chat")

	// WebSocket
	chat.GET("/conversations/:conversationId/ws", h.ServeWS)

	protected := chat.Group("/")
	protected.Use(middlewares.Authenticate())
	{
		// Conversations
		protected.POST("/conversations", h.StartConversation)
		protected.GET("/conversations", h.GetConversations)
		protected.GET("/conversations/dm", h.GetDMConversations)

		// Participants
		protected.GET("/conversations/:conversationId/participants", h.GetParticipants)

		// Messages
		protected.GET("/conversations/:conversationId/messages", h.GetMessages)
		protected.POST("/conversations/:conversationId/messages", h.SendMessage)
		protected.DELETE("/conversations/:conversationId/messages/:messageId", h.DeleteMessage)

		// Threads
		protected.GET("/conversations/:conversationId/messages/:messageId/replies", h.GetReplies)

		// Reactions
		protected.POST("/conversations/:conversationId/messages/:messageId/reactions", h.AddReaction)
		protected.DELETE("/conversations/:conversationId/messages/:messageId/reactions/:emoji", h.RemoveReaction)

	}
}
