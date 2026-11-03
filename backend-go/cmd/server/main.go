package main

import (
	"context"
	"errors"
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

const (
	shutdownDrain   = 15 * time.Second
	sessionShutdown = 10 * time.Second
)

func main() {
	cfg := configs.Load()
	logger.Init(cfg.Server.Environment)

	if err := cfg.Validate(); err != nil {
		log.Fatal().Msg("refusing to start: " + err.Error())
	}

	if err := auth.InitJWT(&cfg.JWT); err != nil {
		log.Fatal().Err(err).Msg("refusing to start: invalid JWT configuration")
	}
	auth.SetCookieSecure(cfg.Security.CookieSecure)

	log.Info().Str("env", cfg.Server.Environment).Msg("Starting DevOps Journey Lab backend")

	if err := docker.InitClient(cfg.Docker.Host); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Docker client")
	}
	defer docker.Close()

	if err := database.Connect(&cfg.Database); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	if err := database.SeedAdminUser(); err != nil {
		log.Error().Err(err).Msg("Failed to seed admin user")
	}

	if err := database.ConnectRedis(&cfg.Redis); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer database.CloseRedis()

	if err := database.OptimizeConnectionPool(); err != nil {
		log.Fatal().Err(err).Msg("Failed to optimize connection pool")
	}

	if err := task.Init(cfg.Docker.TasksDir); err != nil {
		log.Fatal().Err(err).Msg("Failed to load tasks")
	}

	websocket.InitUpgrader(cfg.CORS.Origins)

	sessionManager := session.NewManager(&cfg.Docker)
	// The expiry reaper API is owned by the session package; start it if the manager provides one.
	if reaper, ok := any(sessionManager).(interface{ StartPeriodicCleanup(time.Duration) }); ok {
		reaper.StartPeriodicCleanup(5 * time.Minute)
	} else {
		log.Warn().Msg("session manager has no StartPeriodicCleanup; relying on per-session expiry timers")
	}

	poolMonitor := database.NewPoolMonitor(30 * time.Second)
	poolMonitor.Start()
	defer poolMonitor.Stop()

	api.Init(sessionManager)

	if cfg.Server.GinMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(cfg.Server.GinMode)
	}

	router := gin.New()
	// Only these proxies may set X-Forwarded-For; otherwise ClientIP() (rate limits, audit log)
	// would be spoofable by any client.
	if err := router.SetTrustedProxies(cfg.Security.TrustedProxies); err != nil {
		log.Fatal().Err(err).Strs("trusted_proxies", cfg.Security.TrustedProxies).Msg("Invalid TRUSTED_PROXIES")
	}
	router.ForwardedByClientIP = true
	router.RemoteIPHeaders = []string{"X-Forwarded-For", "X-Real-IP"}

	api.SetupRoutes(router, cfg.CORS.Origins)

	srv := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// 0: long-lived WebSocket terminals manage their own deadlines (gorilla clears the
		// server deadlines on upgrade and the handler sets per-write deadlines).
		WriteTimeout:   0,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	exitCode := 0
	select {
	case <-sigCtx.Done():
		log.Info().Msg("Shutdown signal received, draining connections")
	case err := <-serverErr:
		log.Error().Err(err).Msg("HTTP server failed")
		exitCode = 1
	}
	stop() // a second signal now terminates immediately

	ctx, cancel := context.WithTimeout(context.Background(), shutdownDrain)
	defer cancel()

	// 1. Stop accepting new connections and drain in-flight HTTP requests.
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("HTTP drain did not finish in time")
		exitCode = 1
	}

	// 2. Hijacked WebSocket connections are not tracked by Shutdown: close them explicitly
	//    and give the handlers a moment to stop their docker exec sessions.
	if n := websocket.CloseAll("server shutting down"); n > 0 {
		log.Info().Int("count", n).Msg("Closed terminal sockets")
		for websocket.ActiveConnections() > 0 && ctx.Err() == nil {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 3. Stop session timers/cleanup.
	if err := sessionManager.Shutdown(sessionShutdown); err != nil {
		log.Error().Err(err).Msg("Session manager shutdown error")
	}

	log.Info().Msg("Server stopped")
	if exitCode != 0 {
		// Run deferred cleanups (DB, Redis, Docker) before exiting non-zero.
		func() {
			poolMonitor.Stop()
			_ = database.CloseRedis()
			_ = database.Close()
			_ = docker.Close()
		}()
		os.Exit(exitCode)
	}
}
