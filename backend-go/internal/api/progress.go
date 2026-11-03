package api

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// pointsCTE renders "pts(task_id, points)" for the currently loaded tasks so SQL can
// weight progress by task points (points live in task.yaml, not in the database).
func pointsCTE(tasks []models.Task) (string, []any) {
	if len(tasks) == 0 {
		return "pts(task_id, points) AS (SELECT NULL::text, 0 WHERE false)", nil
	}
	var b strings.Builder
	args := make([]any, 0, 2*len(tasks))
	b.WriteString("pts(task_id, points) AS (VALUES ")
	for i, t := range tasks {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(?::text, ?::int)")
		args = append(args, t.ID, t.Points)
	}
	b.WriteString(")")
	return b.String(), args
}

// GetUserProgress handles GET /api/progress.
func GetUserProgress(c *gin.Context) {
	userID, _, _, _ := currentUser(c)
	resp, err := buildProgress(c.Request.Context(), userID, nowUTC())
	if err != nil {
		internalError(c, err, "failed to build progress")
		return
	}
	c.JSON(http.StatusOK, resp)
}

func buildProgress(ctx context.Context, userID uint, now time.Time) (*models.ProgressResponse, error) {
	db := database.DB.WithContext(ctx)
	tasks := task.GetTasks()
	order := make(map[string]int, len(tasks))
	points := make(map[string]int, len(tasks))
	for i, t := range tasks {
		order[t.ID] = i
		points[t.ID] = t.Points
	}

	var rows []models.Progress
	if err := db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}

	resp := &models.ProgressResponse{
		Progress:   []models.ProgressItem{},
		TotalTasks: len(tasks),
	}
	completed := map[string]bool{}
	var completions []time.Time
	for _, p := range rows {
		if _, known := order[p.TaskID]; !known {
			continue // task was removed from tasks/
		}
		resp.Progress = append(resp.Progress, models.ProgressItem{
			TaskID: p.TaskID, Score: p.Score, Total: p.Total, Passed: p.Passed,
			Attempts: p.Attempts, TimeSpent: p.TimeSpent, CompletedAt: p.CompletedAt,
		})
		if p.Passed {
			completed[p.TaskID] = true
			resp.Completed++
			resp.TotalScore += points[p.TaskID]
			if p.CompletedAt != nil {
				completions = append(completions, *p.CompletedAt)
			}
		}
	}
	sort.Slice(resp.Progress, func(i, j int) bool {
		return order[resp.Progress[i].TaskID] < order[resp.Progress[j].TaskID]
	})
	for _, t := range tasks {
		resp.TotalPossible += t.Points
	}
	resp.Percentage = percent(resp.TotalScore, resp.TotalPossible)
	resp.Skills = computeSkills(tasks, completed)

	// Time in lab: closed sessions count ended_at - created_at; active ones count until now.
	// Legacy rows without ended_at fall back to the last check time. Always capped at expires_at.
	var secs float64
	if err := db.Raw(`
		SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (end_t - created_at))), 0)
		FROM (
			SELECT created_at,
			       LEAST(expires_at, COALESCE(ended_at,
			             CASE WHEN status = ? THEN ?::timestamptz ELSE GREATEST(last_check_at, created_at) END)) AS end_t
			FROM sessions WHERE user_id = ?
		) s
		WHERE end_t > created_at`, models.SessionActive, now, userID).Scan(&secs).Error; err != nil {
		return nil, err
	}
	resp.TimeInLabSeconds = int64(secs)

	// Activity days: any check attempt or completion (bounded to ~13 months).
	var days []string
	if err := db.Raw(`
		SELECT DISTINCT d FROM (
			SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS d
			FROM check_attempts WHERE user_id = ? AND created_at >= ?
			UNION
			SELECT to_char(completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD')
			FROM progresses WHERE user_id = ? AND completed_at IS NOT NULL AND completed_at >= ?
		) x`, userID, now.AddDate(0, -13, 0), userID, now.AddDate(0, -13, 0)).Scan(&days).Error; err != nil {
		return nil, err
	}
	resp.StreakDays = computeStreak(days, now)

	firstAttempt, firstPass, err := loadFirstAttempts(db, userID)
	if err != nil {
		return nil, err
	}
	resp.Achievements = computeAchievements(achievementInput{
		TotalTasks:   len(tasks),
		Completions:  completions,
		ActivityDays: days,
		FirstAttempt: firstAttempt,
		FirstPass:    firstPass,
	})
	return resp, nil
}

// loadFirstAttempts returns the first attempt and the first passing attempt per task.
func loadFirstAttempts(db *gorm.DB, userID uint) ([]attemptRecord, []attemptRecord, error) {
	var first, pass []attemptRecord
	if err := db.Raw(`
		SELECT DISTINCT ON (task_id) task_id, passed, elapsed_seconds, created_at
		FROM check_attempts WHERE user_id = ?
		ORDER BY task_id, created_at, id`, userID).Scan(&first).Error; err != nil {
		return nil, nil, err
	}
	if err := db.Raw(`
		SELECT DISTINCT ON (task_id) task_id, passed, elapsed_seconds, created_at
		FROM check_attempts WHERE user_id = ? AND passed
		ORDER BY task_id, created_at, id`, userID).Scan(&pass).Error; err != nil {
		return nil, nil, err
	}
	return first, pass, nil
}

// ResetProgress handles DELETE /api/progress (resets the caller's own progress).
func ResetProgress(c *gin.Context) {
	userID, _, _, _ := currentUser(c)
	err := database.WithTransaction(c.Request.Context(), func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.CheckAttempt{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&models.Progress{}).Error
	})
	if err != nil {
		internalError(c, err, "failed to reset progress")
		return
	}
	logSelf(c, models.LevelWarn, "progress_reset", "User reset own progress")
	c.Status(http.StatusNoContent)
}

// GetLeaderboard handles GET /api/leaderboard?period=week|month|all&limit=1..100.
func GetLeaderboard(c *gin.Context) {
	now := nowUTC()
	since, ok := periodStart(c.Query("period"), now)
	if !ok {
		respondError(c, http.StatusBadRequest, "Invalid period", "period must be week, month or all")
		return
	}
	limit, ok := parseLimit(c, 50, 1, 100)
	if !ok {
		return
	}

	rows, err := leaderboardRows(c.Request.Context(), since)
	if err != nil {
		internalError(c, err, "failed to load leaderboard")
		return
	}
	userID, username, _, _ := currentUser(c)
	c.JSON(http.StatusOK, buildLeaderboard(rows, limit, userID, username))
}

// leaderboardRows aggregates points of tasks completed since the given time for active users.
func leaderboardRows(ctx context.Context, since time.Time) ([]leaderboardRow, error) {
	cte, args := pointsCTE(task.GetTasks())
	args = append(args, since)
	var rows []leaderboardRow
	err := database.DB.WithContext(ctx).Raw(`
		WITH `+cte+`
		SELECT u.id AS user_id, u.username,
		       SUM(pts.points) AS score, COUNT(*) AS completed, MAX(p.completed_at) AS last_completed
		FROM progresses p
		JOIN pts ON pts.task_id = p.task_id
		JOIN users u ON u.id = p.user_id
		WHERE p.passed AND p.completed_at IS NOT NULL AND p.completed_at >= ?
		  AND u.active AND u.deleted_at IS NULL
		GROUP BY u.id, u.username`, args...).Scan(&rows).Error
	return rows, err
}
