package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/api"
	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/devops-lab/backend/internal/websocket"
	"github.com/devops-lab/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// @title DevOps Lab API
// @version 2.0
// @description Enterprise DevOps Learning Platform API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.devopslab.io/support
// @contact.email support@devopslab.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// Load configuration
	cfg := configs.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Initialize logger
	logger.Init(cfg.Server.Environment)

	log.Info().Msg("🚀 Starting DevOps Lab Backend v2.0")

	// Initialize Docker client
	if err := docker.InitClient(cfg.Docker.Socket); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Docker client")
	}
	defer docker.Close()

	// Connect to PostgreSQL
	if err := database.Connect(&cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Connect to Redis
	if err := database.ConnectRedis(&cfg.Redis); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer database.CloseRedis()

	// Optimize connection pool
	if err := database.OptimizeConnectionPool(); err != nil {
		log.Fatal().Err(err).Msg("Failed to optimize connection pool")
	}

	// Initialize JWT
	auth.InitJWT(&cfg.JWT)

	// Load tasks
	if err := task.Init(cfg.Docker.TasksDir); err != nil {
		log.Fatal().Err(err).Msg("Failed to load tasks")
	}

	// Initialize WebSocket CORS
	websocket.InitUpgrader(cfg.CORS.Origins)
	log.Info().Strs("allowed_origins", cfg.CORS.Origins).Msg("WebSocket CORS configured")

	// Initialize session manager
	sessionManager := session.NewManager(&cfg.Docker)

	// Start periodic cleanup with CleanupManager
	sessionManager.StartPeriodicCleanup(5 * time.Minute)
	log.Info().Msg("Session cleanup manager started")

	// Start connection pool monitor
	poolMonitor := database.NewPoolMonitor(30 * time.Second)
	poolMonitor.Start()
	log.Info().Msg("Connection pool monitor started")

	// Initialize API handlers
	api.Init(sessionManager)

	// Setup Gin
	if cfg.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	api.SetupRoutes(router, cfg.CORS.Origins)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop pool monitor
	poolMonitor.Stop()
	log.Info().Msg("Connection pool monitor stopped")

	// Shutdown session manager (CleanupManager)
	if err := sessionManager.Shutdown(10 * time.Second); err != nil {
		log.Error().Err(err).Msg("Session manager shutdown error")
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped gracefully")
}
