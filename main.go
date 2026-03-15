package main

import (
	"colossa-pm/authentication"
	"colossa-pm/database"
	"colossa-pm/logger"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var dbConn database.DbConnection

func initlizeApp(route *gin.Engine) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbConn := database.Instance()

	if err := dbConn.InitializeDb(); err != nil {
		log.Fatal("Failed to connect to database")
	}

	db := dbConn.DB()

	v1 := route.Group("/api/v1")

	authentication.RegisterRoutes(v1, db)

}

func main() {
	route := gin.New()

	logger := logger.Instance()

	route.Use(logger.GinEndpointLogger())
	route.Use(cors.Default())

	initlizeApp(route)
	// defer dbConn.DisconnectDb()

	route.GET("/_health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "Success",
			"message": "Success",
			"data":    gin.H{},
		})
	})

	route.Run(":3002")
}
