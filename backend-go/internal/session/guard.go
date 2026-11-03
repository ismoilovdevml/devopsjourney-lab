package session

import (
	"context"
	"fmt"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
)

// Resource guards: limits a lab container cannot enforce on itself.
//
//   - Disk: the writable layer of a lab container has no size limit unless
//     LAB_STORAGE_SIZE is usable (overlay2 on xfs with pquota). The sweeper
//     therefore measures SizeRw every minute and stops sessions above
//     LAB_DISK_MAX_BYTES. A student can overshoot by what they write in one
//     interval; the host needs that much headroom per session.
//   - Idle: a session nobody has a terminal open to still holds one of the
//     MAX_SESSIONS_TOTAL slots. Sessions without an attached terminal for
//     LAB_IDLE_TIMEOUT are stopped like an expiry.

// ActivityFunc reports whether a terminal is attached to a session right now
// and when the session's terminal was last used (zero if never, in this
// process).
type ActivityFunc func(sessionID string) (attached bool, lastActivity time.Time)

// Reasons recorded in sessions.end_reason.
const (
	ReasonDiskQuota = "disk_quota"
	ReasonIdle      = "idle"
)

// SetActivitySource wires the terminal activity registry (the websocket
// package) into idle detection. Without it, idle reclaim is disabled.
func (m *Manager) SetActivitySource(f ActivityFunc) {
	m.activityMu.Lock()
	m.activity = f
	m.activityMu.Unlock()
}

func (m *Manager) activitySource() ActivityFunc {
	m.activityMu.RLock()
	defer m.activityMu.RUnlock()
	return m.activity
}

// TaskAvailability reports whether the task can run on this server and, if
// not, a reason suitable for end users.
func (m *Manager) TaskAvailability(taskID string) (bool, string) {
	rt, err := loadTaskRuntime(m.cfg.TasksDir, taskID)
	if err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("Task runtime config unreadable; reporting task unavailable")
		return false, "This lab is misconfigured on the server."
	}
	return taskAvailable(rt, m.cfg.AllowNetAdmin)
}

// taskAvailable is the pure availability rule.
func taskAvailable(rt *TaskRuntime, allowNetAdmin bool) (bool, string) {
	if !allowNetAdmin && rt.needsCapability(docker.CapNetAdmin) {
		return false, "This lab needs network-admin privileges (CAP_NET_ADMIN), which this server does not grant."
	}
	return true, ""
}

// StartResourceGuards runs the disk-quota and idle checks every interval
// (keep it at a minute or less: it bounds how far past the quota a student
// can get).
func (m *Manager) StartResourceGuards(interval time.Duration) {
	m.cleanupManager.StartPeriodicCleanup(interval, m.EnforceResourceGuards)
}

// EnforceResourceGuards stops sessions over the disk quota or idle for too long.
func (m *Manager) EnforceResourceGuards(ctx context.Context) error {
	var active []models.Session
	if err := database.DB.WithContext(ctx).Where("status = ?", StatusActive).Find(&active).Error; err != nil {
		return fmt.Errorf("failed to list active sessions: %w", err)
	}
	if len(active) == 0 {
		return nil
	}

	ended := make(map[string]bool)

	if m.cfg.DiskMaxBytes > 0 {
		sizes, err := docker.ListLabContainerSizes(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Disk quota check skipped")
		} else {
			for _, v := range diskViolations(active, sizes, m.cfg.DiskMaxBytes) {
				ended[v.session.ID] = true
				detail := fmt.Sprintf("Lab for task %s stopped: container disk usage %s exceeded the %s limit",
					v.session.TaskID, humanBytes(v.sizeRw), humanBytes(m.cfg.DiskMaxBytes))
				log.Warn().Str("session_id", v.session.ID).Int64("size_rw", v.sizeRw).
					Int64("limit", m.cfg.DiskMaxBytes).Msg("Session over disk quota, stopping it")
				m.finishSessionWithReason(ctx, v.session, StatusExpired, ReasonDiskQuota)
				m.audit(ctx, v.session.UserID, models.LevelWarn, "session_disk_quota", detail)
			}
		}
	}

	if f := m.activitySource(); f != nil && m.cfg.IdleTimeout > 0 {
		for _, s := range idleSessions(active, f, time.Now(), m.startedAt, m.cfg.IdleTimeout) {
			if ended[s.ID] {
				continue
			}
			log.Info().Str("session_id", s.ID).Dur("idle_timeout", m.cfg.IdleTimeout).
				Msg("No terminal attached, reclaiming idle session")
			m.finishSessionWithReason(ctx, s, StatusExpired, ReasonIdle)
			m.audit(ctx, s.UserID, models.LevelInfo, "session_idle_reclaimed",
				fmt.Sprintf("Lab for task %s stopped: no terminal attached for %s", s.TaskID, m.cfg.IdleTimeout))
		}
	}
	return nil
}

type diskViolation struct {
	session *models.Session
	sizeRw  int64
}

// diskViolations returns the active sessions whose container writable layer
// is larger than maxBytes. Containers are matched by ID; a container whose
// session label disagrees with the row is ignored (it is the orphan
// sweeper's job).
func diskViolations(active []models.Session, sizes []docker.LabContainer, maxBytes int64) []diskViolation {
	if maxBytes <= 0 {
		return nil
	}
	byContainer := make(map[string]docker.LabContainer, len(sizes))
	for _, c := range sizes {
		byContainer[c.ID] = c
	}
	var out []diskViolation
	for i := range active {
		s := &active[i]
		c, ok := byContainer[s.ContainerID]
		if !ok || c.SessionID != s.ID {
			continue
		}
		if c.SizeRw > maxBytes {
			out = append(out, diskViolation{session: s, sizeRw: c.SizeRw})
		}
	}
	return out
}

// idleSessions returns active sessions with no terminal attached whose last
// sign of life (creation, last terminal activity, or this process' start,
// whichever is latest) is older than timeout.
func idleSessions(active []models.Session, activity ActivityFunc, now, processStart time.Time, timeout time.Duration) []*models.Session {
	var out []*models.Session
	for i := range active {
		s := &active[i]
		attached, last := activity(s.ID)
		if attached {
			continue
		}
		since := latest(s.CreatedAt, last, processStart)
		if now.Sub(since) > timeout {
			out = append(out, s)
		}
	}
	return out
}

func latest(ts ...time.Time) time.Time {
	var out time.Time
	for _, t := range ts {
		if t.After(out) {
			out = t
		}
	}
	return out
}

// audit writes an activity-log entry for a backend-initiated action.
func (m *Manager) audit(ctx context.Context, userID uint, level, action, details string) {
	if database.DB == nil {
		return
	}
	var u models.User
	_ = database.DB.WithContext(ctx).Select("id", "username").Where("id = ?", userID).Take(&u).Error
	entry := models.ActivityLog{
		UserID:    userID,
		Username:  u.Username,
		Action:    action,
		Details:   details,
		IPAddress: "system",
		Level:     level,
	}
	if err := database.DB.WithContext(ctx).Create(&entry).Error; err != nil {
		log.Error().Err(err).Str("action", action).Msg("Failed to write activity log")
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
