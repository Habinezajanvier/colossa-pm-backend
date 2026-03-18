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

	chat.Use(middlewares.Authenticate())
	{
		// Conversations
		chat.POST("/conversations", h.StartConversation)
		chat.GET("/conversations", h.GetConversations)

		// Messages
		chat.GET("/conversations/:conversationId/messages", h.GetMessages)
		chat.POST("/conversations/:conversationId/messages", h.SendMessage)
		chat.DELETE("/conversations/:conversationId/messages/:messageId", h.DeleteMessage)

		// Threads
		chat.GET("/conversations/:conversationId/messages/:messageId/replies", h.GetReplies)

		// Reactions
		chat.POST("/conversations/:conversationId/messages/:messageId/reactions", h.AddReaction)
		chat.DELETE("/conversations/:conversationId/messages/:messageId/reactions/:emoji", h.RemoveReaction)

	}
}
