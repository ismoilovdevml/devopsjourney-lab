package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxAttemptDelta caps the time credited for a single check (guards against clock skew / stale sessions).
const maxAttemptDelta = 24 * time.Hour

func nowUTC() time.Time { return time.Now().UTC() }

// labError maps session-manager errors to HTTP responses; unknown errors become a logged 500.
func labError(c *gin.Context, err error, logMsg string) {
	switch {
	case errors.Is(err, session.ErrCapacity):
		respondError(c, http.StatusServiceUnavailable, "Lab capacity reached", session.ErrCapacity.Error())
	case errors.Is(err, session.ErrSessionNotFound):
		respondError(c, http.StatusNotFound, "Session not found", "")
	case errors.Is(err, session.ErrTaskNotFound):
		respondError(c, http.StatusNotFound, "Task not found", "")
	case errors.Is(err, session.ErrCheckInProgress):
		respondError(c, http.StatusConflict, "Check in progress", session.ErrCheckInProgress.Error())
	case errors.Is(err, session.ErrContainerNotRunning):
		respondError(c, http.StatusConflict, "Lab container not running", "recreate the lab environment and try again")
	default:
		internalError(c, err, logMsg)
	}
}

func toSessionResponse(s *models.Session, now time.Time) models.SessionResponse {
	ttl := int(s.ExpiresAt.Sub(now).Seconds())
	if ttl < 0 {
		ttl = 0
	}
	return models.SessionResponse{
		SessionID:  s.ID,
		TaskID:     s.TaskID,
		TTLSeconds: ttl,
		ExpiresAt:  s.ExpiresAt,
		CreatedAt:  s.CreatedAt,
	}
}

// ownedSession loads the session named by the path param and verifies the caller owns it.
// Missing and foreign sessions both produce 404 so existence is not leaked.
func ownedSession(c *gin.Context, param string) (*models.Session, bool) {
	id := c.Param(param)
	if !sessionIDPattern.MatchString(id) {
		respondError(c, http.StatusNotFound, "Session not found", "")
		return nil, false
	}
	userID, _, _, _ := currentUser(c)
	s, err := sessionManager.GetSession(c.Request.Context(), id)
	if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
		// Infrastructure failure (DB/Redis): do not disguise an outage as "not found".
		labError(c, err, "failed to load session")
		return nil, false
	}
	if err != nil || s == nil || s.UserID != userID {
		respondError(c, http.StatusNotFound, "Session not found", "")
		return nil, false
	}
	return s, true
}

// CreateSession handles POST /api/sessions.
func CreateSession(c *gin.Context) {
	var req models.SessionCreateRequest
	if !bindJSON(c, &req, "taskId is required") {
		return
	}
	if !taskIDPattern.MatchString(req.TaskID) {
		respondError(c, http.StatusBadRequest, "Invalid request", "invalid taskId")
		return
	}
	tasks := task.GetTasks()
	if indexOfTask(tasks, req.TaskID) < 0 {
		respondError(c, http.StatusNotFound, "Task not found", "")
		return
	}

	userID, _, _, _ := currentUser(c)
	st, err := loadTaskState(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err, "failed to load task state")
		return
	}
	if !unlockedSet(tasks, st.completed, isAdmin(c))[req.TaskID] {
		respondError(c, http.StatusConflict, "Task locked", "complete the previous task first")
		return
	}

	resp, err := sessionManager.CreateSession(c.Request.Context(), userID, req.TaskID)
	if err != nil {
		logSelf(c, models.LevelError, "session_start_failed", "Failed to start session for task: "+req.TaskID)
		labError(c, err, "failed to create session")
		return
	}
	logSelf(c, models.LevelInfo, "session_start", "Started session for task: "+req.TaskID)
	c.JSON(http.StatusCreated, resp)
}

// GetActiveSession handles GET /api/sessions/active (204 when none).
func GetActiveSession(c *gin.Context) {
	userID, _, _, _ := currentUser(c)
	s, err := sessionManager.ActiveSessionForUser(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err, "failed to load active session")
		return
	}
	if s == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, toSessionResponse(s, nowUTC()))
}

// GetSession handles GET /api/sessions/:id.
func GetSession(c *gin.Context) {
	s, ok := ownedSession(c, "id")
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toSessionResponse(s, nowUTC()))
}

// RecreateSession handles POST /api/sessions/:id/recreate.
func RecreateSession(c *gin.Context) {
	s, ok := ownedSession(c, "id")
	if !ok {
		return
	}
	resp, err := sessionManager.Recreate(c.Request.Context(), s.ID)
	if err != nil {
		logSelf(c, models.LevelError, "session_recreate_failed", "Failed to recreate session for task: "+s.TaskID)
		labError(c, err, "failed to recreate session")
		return
	}
	logSelf(c, models.LevelInfo, "session_recreate", "Recreated session for task: "+s.TaskID)
	c.JSON(http.StatusCreated, resp)
}

// DeleteSession handles DELETE /api/sessions/:id.
func DeleteSession(c *gin.Context) {
	s, ok := ownedSession(c, "id")
	if !ok {
		return
	}
	if err := sessionManager.DeleteSession(c.Request.Context(), s.ID); err != nil {
		labError(c, err, "failed to delete session")
		return
	}
	logSelf(c, models.LevelInfo, "session_stop", "Stopped session for task: "+s.TaskID)
	c.Status(http.StatusNoContent)
}

// CheckSolution handles POST /api/check/:sessionId.
func CheckSolution(c *gin.Context) {
	s, ok := ownedSession(c, "sessionId")
	if !ok {
		return
	}
	result, err := sessionManager.CheckSolution(c.Request.Context(), s.ID)
	if err != nil || result == nil {
		if err == nil {
			err = errors.New("nil check result")
		}
		logSelf(c, models.LevelError, "check_failed", "Check failed for task: "+s.TaskID)
		labError(c, err, "failed to run checks")
		return
	}

	first, err := recordAttempt(c.Request.Context(), s, result, nowUTC())
	if err != nil {
		// The check itself succeeded; do not hide the result from the user.
		logSelf(c, models.LevelError, "progress_save_failed", "Failed to save progress for task: "+s.TaskID)
		internalError(c, err, "failed to record progress")
		return
	}
	if first {
		logSelf(c, models.LevelInfo, "task_complete",
			fmt.Sprintf("Completed task: %s (score %d/%d)", s.TaskID, result.Score, result.Total))
	}
	c.JSON(http.StatusOK, result)
}

// recordAttempt stores the check attempt and upserts the user's progress row.
// It returns true when this attempt completed the task for the first time.
func recordAttempt(ctx context.Context, s *models.Session, result *models.CheckResponse, now time.Time) (bool, error) {
	first := false
	err := database.WithTransaction(ctx, func(tx *gorm.DB) error {
		// Time credited = since the previous check in this session, or since the session started.
		since := s.CreatedAt
		var prev models.CheckAttempt
		err := tx.Where("session_id = ? AND user_id = ?", s.ID, s.UserID).Order("created_at DESC").Take(&prev).Error
		if err == nil && prev.CreatedAt.After(since) {
			since = prev.CreatedAt
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		delta := clampDuration(now.Sub(since), 0, maxAttemptDelta)
		elapsed := clampDuration(now.Sub(s.CreatedAt), 0, maxAttemptDelta)

		attempt := models.CheckAttempt{
			UserID:         s.UserID,
			TaskID:         s.TaskID,
			SessionID:      s.ID,
			Score:          result.Score,
			Total:          result.Total,
			Passed:         result.Passed,
			ElapsedSeconds: int(elapsed.Seconds()),
			CreatedAt:      now,
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return err
		}

		// Make sure the row exists, then lock it (safe against concurrent checks).
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.Progress{
			UserID: s.UserID, TaskID: s.TaskID, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			return err
		}
		var existing models.Progress
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND task_id = ?", s.UserID, s.TaskID).Take(&existing).Error; err != nil {
			return err
		}

		merged, firstCompletion := mergeProgress(existing, result, int(delta.Seconds()), now)
		first = firstCompletion
		return tx.Model(&models.Progress{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"score":        merged.Score,
			"total":        merged.Total,
			"passed":       merged.Passed,
			"attempts":     merged.Attempts,
			"time_spent":   merged.TimeSpent,
			"completed_at": merged.CompletedAt,
			"updated_at":   now,
		}).Error
	})
	return first, err
}

func clampDuration(d, lo, hi time.Duration) time.Duration {
	return min(max(d, lo), hi)
}

// mergeProgress applies one check result to a progress row: attempts++, best score,
// passed is sticky, completed_at is set on the first pass, time spent accumulates.
func mergeProgress(p models.Progress, r *models.CheckResponse, deltaSeconds int, now time.Time) (models.Progress, bool) {
	p.Attempts++
	if deltaSeconds > 0 {
		p.TimeSpent += deltaSeconds
	}
	if r.Total != p.Total {
		// Task definition changed (or first attempt): best score is relative to the new total.
		if r.Score > p.Score || p.Total == 0 || p.Score > r.Total {
			p.Score = r.Score
		}
		p.Total = r.Total
	} else if r.Score > p.Score {
		p.Score = r.Score
	}
	if p.Score > p.Total {
		p.Score = p.Total
	}
	first := false
	if r.Passed && !p.Passed {
		p.Passed = true
		first = true
	}
	if p.Passed && p.CompletedAt == nil {
		t := now
		p.CompletedAt = &t
	}
	return p, first
}
