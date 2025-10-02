package session

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CleanupManager manages scheduled cleanup tasks with proper goroutine lifecycle
type CleanupManager struct {
	mu      sync.RWMutex
	timers  map[string]*time.Timer
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
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
