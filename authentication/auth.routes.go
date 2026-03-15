package authentication

import (
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
	svc := NewService(repo)
	h := NewHandler(svc)

	authGroup := rg.Group("/auth")
	{
		// Public routes
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/refresh", h.RefreshTokens)

		// Protected routes
		protected := authGroup.Group("/")
		protected.Use(middlewares.Authenticate())
		{
			protected.PUT("/change-password", h.ChangePassword)
		}
	}
}
