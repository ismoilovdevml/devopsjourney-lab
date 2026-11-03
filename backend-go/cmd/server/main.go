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
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/devops-lab/backend/internal/websocket"
	"github.com/devops-lab/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// version is the build version reported by /healthz. Set at build time with
// -ldflags "-X main.version=<version>" (see Dockerfile and Makefile).
var version = "dev"

const (
	shutdownDrain   = 15 * time.Second
	sessionShutdown = 10 * time.Second
	// sessionReapInterval is how often expired sessions and orphaned lab containers are swept.
	sessionReapInterval = 5 * time.Minute
	// resourceGuardInterval is how often lab disk usage and idle sessions are checked; it
	// bounds how far past LAB_DISK_MAX_BYTES a student can write before the lab is stopped.
	resourceGuardInterval = time.Minute
)

// checkDefaultAdminPasswords refuses to start in production while an admin account still
// uses a well-known default password; in development it only warns loudly.
func checkDefaultAdminPasswords(production bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	weak, err := database.AdminsWithDefaultPassword(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Could not check admin accounts for default passwords")
		return
	}
	if len(weak) == 0 {
		return
	}
	msg := "admin account(s) use a well-known default password; log in and change it " +
		"(POST /api/me/password), or reset it in the database, then restart"
	if production {
		log.Fatal().Strs("admins", weak).Msg("refusing to start: " + msg)
	}
	log.Warn().Strs("admins", weak).Msg("SECURITY WARNING: " + msg + " (allowed only outside production)")
}

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

	log.Info().Str("env", cfg.Server.Environment).Str("version", version).Msg("Starting DevOps Journey Lab backend")

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
	checkDefaultAdminPasswords(cfg.IsProduction())

	if err := database.ConnectRedis(&cfg.Redis); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer func() {
		if err := database.CloseRedis(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis client")
		}
	}()

	if err := database.OptimizeConnectionPool(); err != nil {
		log.Fatal().Err(err).Msg("Failed to optimize connection pool")
	}

	if err := task.Init(cfg.Docker.TasksDir); err != nil {
		log.Fatal().Err(err).Msg("Failed to load tasks")
	}

	websocket.InitUpgrader(cfg.CORS.Origins)
	middleware.SetAuthRateLimit(cfg.Security.AuthRateLimitPerMin)
	middleware.SetSessionRateLimit(cfg.Security.SessionRateLimitPerMin)
	if cfg.Security.AuthRateLimitPerMin != configs.DefaultAuthRateLimitPerMin {
		log.Warn().Int("per_minute", cfg.Security.AuthRateLimitPerMin).
			Msg("AUTH_RATE_LIMIT_PER_MIN differs from the production default")
	}

	sessionManager := session.NewManager(&cfg.Docker)
	// Reconciles DB sessions with lab containers now (crash recovery) and every interval.
	sessionManager.StartPeriodicCleanup(sessionReapInterval)
	// Idle reclaim needs to know which sessions have a terminal attached.
	sessionManager.SetActivitySource(websocket.SessionActivity)
	// Disk quota (LAB_DISK_MAX_BYTES) and idle reclaim (LAB_IDLE_TIMEOUT).
	sessionManager.StartResourceGuards(resourceGuardInterval)
	log.Info().Str("lab_runtime", cfg.Docker.Runtime).Bool("allow_net_admin", cfg.Docker.AllowNetAdmin).
		Str("storage_size", cfg.Docker.StorageSize).Int64("disk_max_bytes", cfg.Docker.DiskMaxBytes).
		Dur("idle_timeout", cfg.Docker.IdleTimeout).Msg("Lab sandbox settings")

	poolMonitor := database.NewPoolMonitor(30 * time.Second)
	poolMonitor.Start()
	defer poolMonitor.Stop()

	api.Init(sessionManager, api.Options{Version: version, DefaultImage: cfg.Docker.DefaultImage})

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
