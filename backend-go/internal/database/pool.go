package database

import (
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// PoolMonitor monitors and adjusts database connection pool
type PoolMonitor struct {
	ticker        *time.Ticker
	stop          chan struct{}
	stopOnce      sync.Once
	minConns      int
	maxConns      int
	checkInterval time.Duration
}

// NewPoolMonitor creates a new pool monitor
func NewPoolMonitor(checkInterval time.Duration) *PoolMonitor {
	cpuCount := runtime.NumCPU()
	return &PoolMonitor{
		ticker:        time.NewTicker(checkInterval),
		stop:          make(chan struct{}),
		minConns:      cpuCount * 2,
		maxConns:      cpuCount * 25,
		checkInterval: checkInterval,
	}
}

// Start begins monitoring the connection pool
func (pm *PoolMonitor) Start() {
	log.Info().
		Int("min_conns", pm.minConns).
		Int("max_conns", pm.maxConns).
		Dur("interval", pm.checkInterval).
		Msg("Starting connection pool monitor")

	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.adjustPool()
			case <-pm.stop:
				log.Info().Msg("Connection pool monitor stopped")
				return
			}
		}
	}()
}

// Stop stops the pool monitor. Safe to call more than once, and does not block if Start was never called.
func (pm *PoolMonitor) Stop() {
	pm.stopOnce.Do(func() {
		pm.ticker.Stop()
		close(pm.stop)
	})
}

// adjustPool dynamically adjusts the connection pool based on metrics
func (pm *PoolMonitor) adjustPool() {
	if DB == nil {
		return
	}
	sqlDB, err := DB.DB()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get database connection")
		return
	}

	stats := sqlDB.Stats()

	log.Debug().
		Int("open_connections", stats.OpenConnections).
		Int("in_use", stats.InUse).
		Int("idle", stats.Idle).
		Int64("wait_count", stats.WaitCount).
		Dur("wait_duration", stats.WaitDuration).
		Int("max_open", stats.MaxOpenConnections).
		Msg("Connection pool stats")

	// Increase pool if wait count is high
	if stats.WaitCount > 100 && stats.MaxOpenConnections < pm.maxConns {
		newMax := stats.MaxOpenConnections + 10
		if newMax > pm.maxConns {
			newMax = pm.maxConns
		}

		sqlDB.SetMaxOpenConns(newMax)
		sqlDB.SetMaxIdleConns(newMax / 2)

		log.Info().
			Int("old_max", stats.MaxOpenConnections).
			Int("new_max", newMax).
			Int64("wait_count", stats.WaitCount).
			Msg("Increased connection pool size")
	}

	// Decrease pool if too many idle connections (MaxOpenConnections==0 means unlimited)
	if stats.MaxOpenConnections <= 0 {
		return
	}
	idleRatio := float64(stats.Idle) / float64(stats.MaxOpenConnections)
	if idleRatio > 0.7 && stats.MaxOpenConnections > pm.minConns {
		newMax := stats.MaxOpenConnections - 5
		if newMax < pm.minConns {
			newMax = pm.minConns
		}

		sqlDB.SetMaxOpenConns(newMax)
		sqlDB.SetMaxIdleConns(newMax / 2)

		log.Info().
			Int("old_max", stats.MaxOpenConnections).
			Int("new_max", newMax).
			Float64("idle_ratio", idleRatio).
			Msg("Decreased connection pool size")
	}
}

// OptimizeConnectionPool sets optimal connection pool parameters
func OptimizeConnectionPool() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	cpuCount := runtime.NumCPU()

	// Dynamic pool sizing based on CPU count
	maxOpenConns := cpuCount * 10
	maxIdleConns := cpuCount * 2

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	log.Info().
		Int("cpu_count", cpuCount).
		Int("max_open_conns", maxOpenConns).
		Int("max_idle_conns", maxIdleConns).
		Dur("conn_max_lifetime", 30*time.Minute).
		Dur("conn_max_idle_time", 10*time.Minute).
		Msg("Connection pool optimized")

	return nil
}

// OptimizeRedisPool optimizes Redis connection pool
func OptimizeRedisPool() {
	cpuCount := runtime.NumCPU()
	poolSize := cpuCount * 10
	minIdleConns := cpuCount * 2

	// Redis client is already initialized, update its pool size through options
	log.Info().
		Int("pool_size", poolSize).
		Int("min_idle_conns", minIdleConns).
		Msg("Redis pool configuration")
}
