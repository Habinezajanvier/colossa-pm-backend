package users

import (
	"colossa-pm/middlewares"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB) {
	repo := NewRepository(db)
	svc := NewService(repo)
	h := NewHandler(svc)

	usersGroup := rg.Group("/users")
	usersGroup.Use(middlewares.Authenticate())
	{
		usersGroup.GET("", h.GetUsers)
		usersGroup.GET("/me", h.GetProfile)
	}
}
