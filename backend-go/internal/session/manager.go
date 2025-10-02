package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	cfg            *configs.DockerConfig
	cleanupManager *CleanupManager
}

func NewManager(cfg *configs.DockerConfig) *Manager {
	return &Manager{
		cfg:            cfg,
		cleanupManager: NewCleanupManager(),
	}
}

// CreateSession creates a new container session
func (m *Manager) CreateSession(ctx context.Context, userID uint, taskID string) (*models.SessionResponse, error) {
	// Get task
	taskObj, err := task.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	log.Info().
		Uint("user_id", userID).
		Str("task_id", taskID).
		Msg("Creating session")

	// Ensure image exists
	if err := docker.EnsureImage(ctx, taskObj.Image); err != nil {
		return nil, fmt.Errorf("failed to ensure image: %w", err)
	}

	// Create container
	containerID, err := docker.CreateContainer(ctx, &docker.ContainerConfig{
		Image:       taskObj.Image,
		MemoryLimit: m.cfg.MemLimit,
		CPULimit:    m.cfg.CPUNano,
		PidsLimit:   m.cfg.PidsLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := docker.StartContainer(ctx, containerID); err != nil {
		_ = docker.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Run setup commands
	if err := docker.RunSetupCommands(ctx, containerID, taskObj.Setup); err != nil {
		_ = docker.StopContainer(ctx, containerID, 2)
		_ = docker.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("setup failed: %w", err)
	}

	// Create session record
	sessionID := containerID[:12]
	expiresAt := time.Now().Add(time.Duration(m.cfg.SessionTTL) * time.Second)

	session := &models.Session{
		ID:          sessionID,
		UserID:      userID,
		TaskID:      taskID,
		ContainerID: containerID,
		Status:      "active",
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}

	// Save to database
	if err := database.DB.Create(session).Error; err != nil {
		_ = docker.StopContainer(ctx, containerID, 2)
		_ = docker.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Cache in Redis
	if err := m.cacheSession(ctx, session); err != nil {
		log.Warn().Err(err).Msg("Failed to cache session in Redis")
	}

	// Schedule cleanup using CleanupManager (fixes goroutine leak)
	m.cleanupManager.Schedule(sessionID, time.Duration(m.cfg.SessionTTL)*time.Second, func() {
		ctx := context.Background()
		log.Info().Str("session_id", sessionID).Msg("Session TTL expired, cleaning up")
		if err := m.DeleteSession(ctx, sessionID); err != nil {
			log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to cleanup expired session")
		}
	})

	log.Info().
		Str("session_id", sessionID).
		Str("container_id", containerID[:12]).
		Msg("Session created successfully")

	return &models.SessionResponse{
		SessionID:  sessionID,
		TTLSeconds: m.cfg.SessionTTL,
		ExpiresAt:  expiresAt,
	}, nil
}

// GetSession retrieves a session
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	// Try Redis cache first
	session, err := m.getCachedSession(ctx, sessionID)
	if err == nil && session != nil {
		return session, nil
	}

	// Fallback to database
	var dbSession models.Session
	if err := database.DB.Where("id = ? AND status = ?", sessionID, "active").First(&dbSession).Error; err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if expired
	if time.Now().After(dbSession.ExpiresAt) {
		_ = m.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	return &dbSession, nil
}

// DeleteSession deletes a session and cleans up container
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	log.Info().Str("session_id", sessionID).Msg("Deleting session")

	// Remove scheduled cleanup to prevent duplicate deletion
	m.cleanupManager.Remove(sessionID)

	// Stop and remove container
	_ = docker.StopContainer(ctx, session.ContainerID, 2)
	_ = docker.RemoveContainer(ctx, session.ContainerID)

	// Update database
	if err := database.DB.Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("status", "stopped").Error; err != nil {
		log.Error().Err(err).Msg("Failed to update session status")
	}

	// Remove from Redis
	_ = database.RedisClient.Del(ctx, "session:"+sessionID).Err()

	log.Info().Str("session_id", sessionID).Msg("Session deleted")
	return nil
}

// CheckSolution runs validation checks
func (m *Manager) CheckSolution(ctx context.Context, sessionID string) (*models.CheckResponse, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get task
	taskObj, err := task.GetTask(session.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Verify container is running
	info, err := docker.InspectContainer(ctx, session.ContainerID)
	if err != nil || !info.State.Running {
		return nil, fmt.Errorf("container is not running")
	}

	log.Info().
		Str("session_id", sessionID).
		Int("checks", len(taskObj.Checks)).
		Msg("Running validation checks")

	// Run checks
	results := make([]models.CheckResult, 0, len(taskObj.Checks))
	passedCount := 0

	for i, check := range taskObj.Checks {
		log.Debug().
			Int("check", i+1).
			Str("name", check.Name).
			Msg("Running check")

		result := docker.ExecCommand(ctx, session.ContainerID, check.Cmd, 15*time.Second)

		checkResult := models.CheckResult{
			Name: check.Name,
			OK:   result.ExitCode == 0,
		}

		if len(result.Output) > 500 {
			checkResult.Output = result.Output[:500]
		} else {
			checkResult.Output = result.Output
		}

		results = append(results, checkResult)
		if checkResult.OK {
			passedCount++
		}
	}

	passed := passedCount == len(results)

	// Update last check time
	_ = database.DB.Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("last_check_at", time.Now()).Error

	response := &models.CheckResponse{
		Passed:  passed,
		Score:   passedCount,
		Total:   len(results),
		Results: results,
	}

	log.Info().
		Str("session_id", sessionID).
		Int("score", passedCount).
		Int("total", len(results)).
		Bool("passed", passed).
		Msg("Checks completed")

	return response, nil
}

// Shutdown gracefully shuts down the manager
func (m *Manager) Shutdown(timeout time.Duration) error {
	log.Info().Msg("Shutting down session manager")
	return m.cleanupManager.Shutdown(timeout)
}

// StartPeriodicCleanup starts periodic cleanup of expired sessions
func (m *Manager) StartPeriodicCleanup(interval time.Duration) {
	m.cleanupManager.StartPeriodicCleanup(interval, m.CleanupExpiredSessions)
}

// GetActiveTimers returns the number of active cleanup timers
func (m *Manager) GetActiveTimers() int {
	return m.cleanupManager.GetActiveTimers()
}

func (m *Manager) cacheSession(ctx context.Context, session *models.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ttl := time.Until(session.ExpiresAt)
	return database.RedisClient.Set(ctx, "session:"+session.ID, data, ttl).Err()
}

func (m *Manager) getCachedSession(ctx context.Context, sessionID string) (*models.Session, error) {
	data, err := database.RedisClient.Get(ctx, "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}

	var session models.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// CleanupExpiredSessions cleans up all expired sessions (run periodically)
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
	var sessions []models.Session
	if err := database.DB.
		Where("status = ? AND expires_at < ?", "active", time.Now()).
		Find(&sessions).Error; err != nil {
		return err
	}

	log.Info().Int("count", len(sessions)).Msg("Cleaning up expired sessions")

	for _, session := range sessions {
		if err := m.DeleteSession(ctx, session.ID); err != nil {
			log.Error().Err(err).Str("session_id", session.ID).Msg("Failed to cleanup session")
		}
	}

	return nil
}
