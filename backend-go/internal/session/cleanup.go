package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// CleanupManager manages scheduled cleanup tasks with proper goroutine lifecycle
type CleanupManager struct {
	mu     sync.RWMutex
	timers map[string]*time.Timer
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager() *CleanupManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &CleanupManager{
		timers: make(map[string]*time.Timer),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Schedule schedules a cleanup task for a session
func (cm *CleanupManager) Schedule(sessionID string, duration time.Duration, callback func()) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Cancel existing timer if present
	if timer, exists := cm.timers[sessionID]; exists {
		timer.Stop()
		delete(cm.timers, sessionID)
	}

	// Create new timer
	timer := time.AfterFunc(duration, func() {
		select {
		case <-cm.ctx.Done():
			log.Debug().Str("session_id", sessionID).Msg("Cleanup canceled due to shutdown")
			return
		default:
			log.Info().Str("session_id", sessionID).Msg("Executing scheduled cleanup")
			callback()
			cm.Remove(sessionID)
		}
	})

	cm.timers[sessionID] = timer
	log.Debug().
		Str("session_id", sessionID).
		Dur("duration", duration).
		Msg("Cleanup scheduled")
}

// Remove cancels and removes a scheduled cleanup task
func (cm *CleanupManager) Remove(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if timer, exists := cm.timers[sessionID]; exists {
		timer.Stop()
		delete(cm.timers, sessionID)
		log.Debug().Str("session_id", sessionID).Msg("Cleanup timer removed")
	}
}

// StartPeriodicCleanup starts a periodic cleanup task for expired sessions
func (cm *CleanupManager) StartPeriodicCleanup(interval time.Duration, cleanupFunc func(context.Context) error) {
	cm.wg.Add(1)
	go func() {
		defer cm.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Info().Dur("interval", interval).Msg("Periodic cleanup started")

		for {
			select {
			case <-ticker.C:
				log.Debug().Msg("Running periodic cleanup")
				if err := cleanupFunc(cm.ctx); err != nil {
					log.Error().Err(err).Msg("Periodic cleanup failed")
				}
			case <-cm.ctx.Done():
				log.Info().Msg("Periodic cleanup stopped")
				return
			}
		}
	}()
}

// Shutdown gracefully shuts down the cleanup manager
func (cm *CleanupManager) Shutdown(timeout time.Duration) error {
	log.Info().Msg("Shutting down cleanup manager")

	// Signal all goroutines to stop
	cm.cancel()

	// Stop all timers
	cm.mu.Lock()
	for sessionID, timer := range cm.timers {
		timer.Stop()
		log.Debug().Str("session_id", sessionID).Msg("Timer stopped during shutdown")
	}
	cm.timers = make(map[string]*time.Timer)
	cm.mu.Unlock()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Cleanup manager shutdown complete")
		return nil
	case <-time.After(timeout):
		log.Warn().Msg("Cleanup manager shutdown timed out")
		return context.DeadlineExceeded
	}
}

// GetActiveTimers returns the number of active cleanup timers
func (cm *CleanupManager) GetActiveTimers() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.timers)
}

// StartPeriodicCleanup runs one reconciliation pass right away (to clean up
// after a crash) and then every interval.
func (m *Manager) StartPeriodicCleanup(interval time.Duration) {
	go func() {
		ctx, cancel := context.WithTimeout(m.cleanupManager.ctx, 2*time.Minute)
		defer cancel()
		if err := m.CleanupExpiredSessions(ctx); err != nil {
			log.Error().Err(err).Msg("Startup session reconciliation failed")
		}
	}()
	m.cleanupManager.StartPeriodicCleanup(interval, m.CleanupExpiredSessions)
}

// CleanupExpiredSessions reconciles the database with the Docker host:
//  1. active sessions past their TTL are expired (container removed),
//  2. active sessions whose container vanished are marked stopped,
//  3. labelled containers without a live session (orphans left by a backend
//     crash or a failed create) are removed.
func (m *Manager) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now()

	var expired []models.Session
	if err := database.DB.WithContext(ctx).
		Where("status = ? AND expires_at < ?", StatusActive, now).
		Find(&expired).Error; err != nil {
		return fmt.Errorf("failed to list expired sessions: %w", err)
	}
	for i := range expired {
		m.finishSession(ctx, &expired[i], StatusExpired)
	}

	containers, err := docker.ListLabContainers(ctx)
	if err != nil {
		return err
	}

	var active []models.Session
	if err := database.DB.WithContext(ctx).
		Where("status = ?", StatusActive).
		Find(&active).Error; err != nil {
		return fmt.Errorf("failed to list active sessions: %w", err)
	}

	byContainer := make(map[string]bool, len(containers))
	for _, c := range containers {
		byContainer[c.ID] = c.State == "running"
	}
	var lost int
	for i := range active {
		s := &active[i]
		if byContainer[s.ContainerID] || s.CreatedAt.After(now.Add(-m.orphanGrace)) {
			continue
		}
		if _, labelled := byContainer[s.ContainerID]; !labelled {
			// Pre-label containers: ask Docker directly before giving up on it.
			if info, err := docker.InspectContainer(ctx, s.ContainerID); err == nil && info.State != nil && info.State.Running {
				continue
			}
		}
		lost++
		m.finishSession(ctx, s, StatusStopped)
	}

	liveSessions := make(map[string]bool, len(active))
	for _, s := range active {
		liveSessions[s.ID] = true
	}
	for _, s := range expired {
		delete(liveSessions, s.ID)
	}
	orphans := findOrphans(containers, liveSessions, now, m.orphanGrace)
	for _, c := range orphans {
		log.Warn().Str("container_id", shortID(c.ID)).Str("session_id", c.SessionID).
			Str("state", c.State).Msg("Removing orphaned lab container")
		if err := docker.RemoveContainer(ctx, c.ID); err != nil {
			log.Error().Err(err).Str("container_id", shortID(c.ID)).Msg("Failed to remove orphaned container")
		}
	}

	if len(expired) > 0 || lost > 0 || len(orphans) > 0 {
		log.Info().Int("expired", len(expired)).Int("lost", lost).Int("orphans", len(orphans)).
			Msg("Session reconciliation done")
	}
	return nil
}

// findOrphans returns labelled containers that do not belong to a live
// session. Running containers younger than grace are skipped: they may still
// be in setup, before their session row is written.
func findOrphans(containers []docker.LabContainer, liveSessions map[string]bool, now time.Time, grace time.Duration) []docker.LabContainer {
	var out []docker.LabContainer
	for _, c := range containers {
		if liveSessions[c.SessionID] {
			continue
		}
		young := now.Sub(c.Created) < grace
		if young && (c.State == "running" || c.State == "created") {
			continue
		}
		out = append(out, c)
	}
	return out
}
