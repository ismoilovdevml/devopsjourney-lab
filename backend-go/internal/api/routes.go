package api

import (
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/websocket"
	"github.com/gin-gonic/gin"
)

// MaxBodyBytes caps request bodies.
const MaxBodyBytes = 1 << 20

// SetupRoutes registers middleware and every route of docs/API.md.
func SetupRoutes(router *gin.Engine, corsOrigins []string) {
	router.Use(
		middleware.RequestID(),
		middleware.LoggerMiddleware(),
		gin.Recovery(),
		middleware.SecurityHeaders(),
		middleware.CORSMiddleware(corsOrigins),
		middleware.OriginGuard(corsOrigins),
		middleware.BodySizeLimit(MaxBodyBytes),
	)

	router.GET("/healthz", HealthCheck)
	router.GET("/readyz", ReadyCheckHandler)

	// Terminal stream; the handler authenticates (cookie) and checks Origin + session ownership.
	router.GET("/ws", websocket.Handler)

	api := router.Group("/api")

	authGroup := api.Group("/auth")
	authGroup.POST("/register", middleware.AuthRateLimiter(), Register)
	authGroup.POST("/login", middleware.AuthRateLimiter(), Login)
	authGroup.POST("/logout", middleware.AuthMiddleware(), Logout)

	public := api.Group("", middleware.OptionalAuthMiddleware(), touchLastActive())
	public.GET("/tasks", GetTasks)
	public.GET("/tasks/:id", GetTask)

	user := api.Group("", middleware.AuthMiddleware(), touchLastActive())
	user.GET("/me", GetMe)
	user.PUT("/me", UpdateMe)
	user.POST("/me/password", ChangePassword)

	// Each create/recreate starts a container: bounded per user (SESSION_RATE_LIMIT_PER_MIN).
	sessionLimit := middleware.SessionRateLimiter()
	user.POST("/sessions", sessionLimit, CreateSession)
	user.GET("/sessions/active", GetActiveSession)
	user.GET("/sessions/:id", GetSession)
	user.POST("/sessions/:id/recreate", sessionLimit, RecreateSession)
	user.DELETE("/sessions/:id", DeleteSession)
	user.POST("/check/:sessionId", CheckSolution)

	user.GET("/progress", GetUserProgress)
	user.DELETE("/progress", ResetProgress)
	user.GET("/leaderboard", GetLeaderboard)

	admin := api.Group("/admin", middleware.AuthMiddleware(), middleware.RoleMiddleware("admin"), touchLastActive())
	admin.GET("/stats", GetStats)
	admin.GET("/users", GetUsers)
	admin.PUT("/users/:id", UpdateUser)
	admin.DELETE("/users/:id", DeleteUserByID)
	admin.GET("/sessions", GetAllSessions)
	admin.POST("/sessions/:id/terminate", TerminateSession)
	admin.GET("/activity", GetActivityLogs)
	admin.GET("/resources", GetResources)
	admin.POST("/tasks/reload", ReloadTasks)
}
