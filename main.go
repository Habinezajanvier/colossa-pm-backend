package main

import (
	"colossa-pm/audit"
	"colossa-pm/authentication"
	"colossa-pm/chat"
	"colossa-pm/database"
	"colossa-pm/logger"
	"colossa-pm/users"
	"colossa-pm/workspace"
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

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		ginMode = gin.DebugMode
	}
	gin.SetMode(ginMode)

	// Initialize DB first — everything depends on it
	dbConn := database.Instance()
	if err := dbConn.InitializeDb(); err != nil {
		log.Fatal("Failed to connect to database")
	}

	db := dbConn.DB()

	route := gin.New()

	appLogger := logger.Instance()
	route.Use(gin.Recovery())
	route.Use(appLogger.GinEndpointLogger())
	route.Use(cors.Default())

	auditRepo := audit.NewAuditRepository(db)
	route.Use(audit.Middleware(audit.NewService(auditRepo)))

	route.Static("/uploads", "./uploads")

	v1 := route.Group("/api/v1")
	audit.RegisterRoutes(v1, auditRepo)
	authentication.RegisterRoutes(v1, db)
	users.RegisterRoutes(v1, db)
	workspace.RegisterRoutes(v1, db)
	chat.RegisterRoutes(v1, db)

	route.GET("/_health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "Success",
			"message": "Success",
			"data":    gin.H{},
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: route,
	}

	go func() {
		appLogger.Log("server starting on port " + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Log("shutdown signal received, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.ErrorMsg("server forced to shutdown: " + err.Error())
	}

	if err := dbConn.DisconnectDb(); err != nil {
		appLogger.ErrorMsg("error closing database: " + err.Error())
	}

	appLogger.Log("shutdown complete")
}
