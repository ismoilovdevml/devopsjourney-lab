package api

import (
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/websocket"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine, corsOrigins []string) {
	// Middleware
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware(corsOrigins))
	router.Use(gin.Recovery())

	// Health check
	router.GET("/healthz", HealthCheck)

	// WebSocket
	router.GET("/ws", websocket.Handler)

	// API v1
	api := router.Group("/api")
	{
		// Auth endpoints (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", Register)
			auth.POST("/login", Login)
		}

		// Tasks (public)
		api.GET("/tasks", GetTasks)
		api.GET("/tasks/:id", GetTask)

		// Protected endpoints
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Sessions
			protected.POST("/sessions", CreateSession)
			protected.GET("/sessions/:id", GetSession)
			protected.DELETE("/sessions/:id", DeleteSession)

			// Check solution
			protected.POST("/check/:id", CheckSolution)

			// User progress
			protected.GET("/progress", GetUserProgress)
			protected.GET("/leaderboard", GetLeaderboard)
		}

		// Admin endpoints
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.RoleMiddleware("admin"))
		{
			admin.GET("/stats", GetStats)
			admin.GET("/users", GetUsers)
			admin.PUT("/users/:id", UpdateUser)
			admin.POST("/tasks/reload", ReloadTasks)
		}
	}
}
