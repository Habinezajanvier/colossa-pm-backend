package main

import (
	"colossa-pm/authentication"
	"colossa-pm/database"
	"colossa-pm/logger"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	route.GET("/_health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "Success",
			"message": "Success",
			"data":    gin.H{},
		})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: route,
	}

	// Start the server in a goroutine so it doesn't block the shutdown logic below
	go func() {
		logger.Log("server starting on port " + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	// Block until we receive SIGINT or SIGTERM (Ctrl+C, kill, Docker stop, etc.)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log("shutdown signal received, shutting down gracefully...")

	// Give in-flight requests up to 10 seconds to complete before closing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.ErrorMsg("server forced to shutdown: " + err.Error())
	}

	// Safely close the DB — all requests have finished
	if err := dbConn.DisconnectDb(); err != nil {
		logger.ErrorMsg("error closing database: " + err.Error())
	}

	logger.Log("shutdown complete")
}
