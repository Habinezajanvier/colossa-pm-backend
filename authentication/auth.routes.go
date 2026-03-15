package authentication

import (
	"colossa-pm/email"
	"colossa-pm/messaging"
	"colossa-pm/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes wires up all auth routes onto the provided router group.
//
// Usage in main.go:
//
//	api := r.Group("/api/v1")
//	auth.RegisterRoutes(api, db)
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	tokenRepo := NewTokenRepository(db)
	mailer := email.NewMailer()
	messageRepo := messaging.NewRepository(db)
	svc := NewService(repo, tokenRepo, mailer, messageRepo)
	h := NewHandler(svc)

	authGroup := rg.Group("/auth")
	{
		// Public routes
		authGroup.POST("/register", h.Register)
		authGroup.POST("/verify-email/:userId", h.VerifyEmail)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.RefreshTokens)
		authGroup.POST("/forget-password", h.RequestChangePassword)
		authGroup.POST("/change-password/confirm", h.ConfirmChangePassword)

		// Protected routes
		protected := authGroup.Group("/")
		protected.Use(middlewares.Authenticate())
		{
			// protected.PUT("/change-password", h.ChangePassword)
		}
	}
}
