package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// ReadyCheck reports whether a dependency is reachable.
type ReadyCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// readyChecks is a variable so tests (and main) can replace it.
var readyChecks = []ReadyCheck{
	{Name: "database", Check: pingDatabase},
	{Name: "redis", Check: pingRedis},
	{Name: "docker", Check: pingDocker},
}

// SetReadyChecks overrides the readiness checks used by /readyz.
func SetReadyChecks(checks []ReadyCheck) {
	readyChecks = checks
}

func pingDatabase(ctx context.Context) error {
	if database.DB == nil {
		return errors.New("not initialized")
	}
	sqlDB, err := database.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func pingRedis(ctx context.Context) error {
	if database.RedisClient == nil {
		return errors.New("not initialized")
	}
	return database.RedisClient.Ping(ctx).Err()
}

func pingDocker(ctx context.Context) error {
	return docker.Ping(ctx)
}

// HealthCheck is the liveness probe: GET /healthz.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Version:   appVersion,
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	})
}

// ReadyCheckHandler is the readiness probe: GET /readyz (200 when DB, Redis and Docker are reachable).
func ReadyCheckHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checks := gin.H{}
	ready := true
	for _, rc := range readyChecks {
		if err := rc.Check(ctx); err != nil {
			log.Warn().Err(err).Str("dependency", rc.Name).Msg("readiness check failed")
			checks[rc.Name] = "down"
			ready = false
			continue
		}
		checks[rc.Name] = "ok"
	}

	status := http.StatusOK
	state := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		state = "not_ready"
	}
	c.JSON(status, gin.H{"status": state, "checks": checks, "timestamp": time.Now().UTC()})
}
