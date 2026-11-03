package api

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// activeUserWindow defines "active_users" in the admin stats.
const activeUserWindow = 24 * time.Hour

// GetStats handles GET /api/admin/stats.
func GetStats(c *gin.Context) {
	db := database.DB.WithContext(c.Request.Context())
	now := nowUTC()
	var s models.StatsResponse

	if err := db.Model(&models.User{}).Count(&s.TotalUsers).Error; err != nil {
		internalError(c, err, "stats: users")
		return
	}
	if err := db.Model(&models.User{}).Where("active AND last_active_at >= ?", now.Add(-activeUserWindow)).
		Count(&s.ActiveUsers).Error; err != nil {
		internalError(c, err, "stats: active users")
		return
	}
	if err := db.Model(&models.Session{}).Where("status = ? AND expires_at > ?", models.SessionActive, now).
		Count(&s.ActiveSessions).Error; err != nil {
		internalError(c, err, "stats: sessions")
		return
	}
	if err := db.Model(&models.CheckAttempt{}).Count(&s.TotalAttempts).Error; err != nil {
		internalError(c, err, "stats: attempts")
		return
	}

	// Per-task attempts / passed users in one grouped query.
	var perTask []struct {
		TaskID   string
		Attempts int64
		Passed   int64
		Started  int64
	}
	if err := db.Model(&models.Progress{}).
		Select("task_id, COALESCE(SUM(attempts), 0) AS attempts, COUNT(*) FILTER (WHERE passed) AS passed, COUNT(*) AS started").
		Group("task_id").Scan(&perTask).Error; err != nil {
		internalError(c, err, "stats: task completion")
		return
	}
	byTask := make(map[string]int, len(perTask))
	for i, r := range perTask {
		byTask[r.TaskID] = i
	}
	var started, passed int64
	s.TaskCompletion = []models.TaskCompletion{}
	for _, t := range task.GetTasks() {
		tc := models.TaskCompletion{TaskID: t.ID}
		if i, ok := byTask[t.ID]; ok {
			tc.Attempts, tc.Passed = perTask[i].Attempts, perTask[i].Passed
			started += perTask[i].Started
			passed += perTask[i].Passed
		}
		s.TaskCompletion = append(s.TaskCompletion, tc)
	}
	// completion_rate: % of started (user, task) pairs that were passed.
	if started > 0 {
		s.CompletionRate = math.Round(1000*float64(passed)/float64(started)) / 10
	}

	var hours []hourCount
	if err := db.Model(&models.Session{}).
		Select("FLOOR(EXTRACT(EPOCH FROM created_at) / 3600)::bigint AS hour, COUNT(*) AS count").
		Where("created_at >= ?", now.Truncate(time.Hour).Add(-23*time.Hour)).
		Group("hour").Scan(&hours).Error; err != nil {
		internalError(c, err, "stats: sessions 24h")
		return
	}
	s.Sessions24h = bucketSessions(hours, now)

	c.JSON(http.StatusOK, s)
}

// GetUsers handles GET /api/admin/users?limit=&offset=.
func GetUsers(c *gin.Context) {
	limit, ok := parseLimit(c, 500, 1, 1000)
	if !ok {
		return
	}
	offset, ok := parseOffset(c)
	if !ok {
		return
	}
	cte, args := pointsCTE(task.GetTasks())
	args = append(args, limit, offset)

	var rows []struct {
		ID           uint
		Username     string
		Email        string
		Role         string
		Active       bool
		Github       string
		Goal         string
		CreatedAt    time.Time
		LastActiveAt *time.Time
		Score        int
		Completed    int
	}
	err := database.DB.WithContext(c.Request.Context()).Raw(`
		WITH `+cte+`,
		agg AS (
			SELECT p.user_id, SUM(pts.points) AS score, COUNT(*) AS completed
			FROM progresses p JOIN pts ON pts.task_id = p.task_id
			WHERE p.passed GROUP BY p.user_id
		)
		SELECT u.id, u.username, u.email, u.role, u.active, u.github, u.goal, u.created_at, u.last_active_at,
		       COALESCE(agg.score, 0) AS score, COALESCE(agg.completed, 0) AS completed
		FROM users u LEFT JOIN agg ON agg.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.id
		LIMIT ? OFFSET ?`, args...).Scan(&rows).Error
	if err != nil {
		internalError(c, err, "failed to list users")
		return
	}

	out := make([]models.AdminUser, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.AdminUser{
			UserResponse: models.UserResponse{
				ID: r.ID, Username: r.Username, Email: r.Email, Role: r.Role, Active: r.Active,
				Github: r.Github, Goal: r.Goal, CreatedAt: r.CreatedAt,
			},
			Score: r.Score, Completed: r.Completed, LastActive: r.LastActiveAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// parseUserID parses the :id path param.
func parseUserID(c *gin.Context) (uint, bool) {
	n, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || n == 0 {
		respondError(c, http.StatusBadRequest, "Invalid user id", "")
		return 0, false
	}
	return uint(n), true
}

// stopUserSessions terminates the user's active lab session, if any (best effort).
func stopUserSessions(c *gin.Context, userID uint) {
	s, err := sessionManager.ActiveSessionForUser(c.Request.Context(), userID)
	if err != nil {
		log.Warn().Err(err).Uint("user_id", userID).Msg("failed to look up active session")
		return
	}
	if s != nil {
		if err := sessionManager.DeleteSession(c.Request.Context(), s.ID); err != nil {
			log.Warn().Err(err).Str("session_id", s.ID).Msg("failed to stop session")
		}
	}
}

// UpdateUser handles PUT /api/admin/users/:id ({role?, active?}).
func UpdateUser(c *gin.Context) {
	targetID, ok := parseUserID(c)
	if !ok {
		return
	}
	var req models.AdminUpdateUserRequest
	if !bindJSON(c, &req, "role must be student or admin; active must be a boolean") {
		return
	}
	selfID, _, _, _ := currentUser(c)
	if targetID == selfID {
		if req.Role != nil && *req.Role != models.RoleAdmin {
			respondError(c, http.StatusBadRequest, "Invalid request", "you cannot demote yourself")
			return
		}
		if req.Active != nil && !*req.Active {
			respondError(c, http.StatusBadRequest, "Invalid request", "you cannot deactivate yourself")
			return
		}
	}

	db := database.DB.WithContext(c.Request.Context())
	var user models.User
	if err := db.Take(&user, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "User not found", "")
			return
		}
		internalError(c, err, "failed to load user")
		return
	}

	updates := map[string]any{}
	if req.Role != nil && *req.Role != user.Role {
		updates["role"] = *req.Role
	}
	if req.Active != nil && *req.Active != user.Active {
		updates["active"] = *req.Active
	}
	if len(updates) > 0 {
		if err := db.Model(&user).Updates(updates).Error; err != nil {
			internalError(c, err, "failed to update user")
			return
		}
		if v, ok := updates["role"].(string); ok {
			user.Role = v
		}
		if v, ok := updates["active"].(bool); ok {
			user.Active = v
		}
		middleware.InvalidateUserCache(user.ID)
		if !user.Active {
			stopUserSessions(c, user.ID)
		}
		logSelf(c, models.LevelWarn, "admin_update_user",
			fmt.Sprintf("Updated user %s (id %d): %v", user.Username, user.ID, updates))
	}
	c.JSON(http.StatusOK, models.NewUserResponse(&user))
}

// DeleteUserByID handles DELETE /api/admin/users/:id.
func DeleteUserByID(c *gin.Context) {
	targetID, ok := parseUserID(c)
	if !ok {
		return
	}
	selfID, _, _, _ := currentUser(c)
	if targetID == selfID {
		respondError(c, http.StatusBadRequest, "Invalid request", "you cannot delete yourself")
		return
	}

	var user models.User
	if err := database.DB.WithContext(c.Request.Context()).Take(&user, targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "User not found", "")
			return
		}
		internalError(c, err, "failed to load user")
		return
	}

	stopUserSessions(c, user.ID)
	// Activity logs are kept for audit (they carry the username).
	err := database.WithTransaction(c.Request.Context(), func(tx *gorm.DB) error {
		for _, m := range []any{&models.CheckAttempt{}, &models.Progress{}, &models.Session{}} {
			if err := tx.Where("user_id = ?", user.ID).Delete(m).Error; err != nil {
				return err
			}
		}
		return tx.Unscoped().Delete(&models.User{}, user.ID).Error
	})
	if err != nil {
		internalError(c, err, "failed to delete user")
		return
	}
	middleware.InvalidateUserCache(user.ID)
	logSelf(c, models.LevelWarn, "admin_delete_user", fmt.Sprintf("Deleted user %s (id %d)", user.Username, user.ID))
	c.Status(http.StatusNoContent)
}

// GetAllSessions handles GET /api/admin/sessions (active sessions, newest first).
func GetAllSessions(c *gin.Context) {
	limit, ok := parseLimit(c, 200, 1, 1000)
	if !ok {
		return
	}
	out := []models.AdminSession{}
	err := database.DB.WithContext(c.Request.Context()).Table("sessions s").
		Select("s.id, s.user_id, u.username, s.task_id, s.status, s.created_at, s.expires_at").
		Joins("JOIN users u ON u.id = s.user_id").
		Where("s.status = ? AND s.expires_at > ?", models.SessionActive, nowUTC()).
		Order("s.created_at DESC").Limit(limit).Scan(&out).Error
	if err != nil {
		internalError(c, err, "failed to list sessions")
		return
	}
	c.JSON(http.StatusOK, out)
}

// TerminateSession handles POST /api/admin/sessions/:id/terminate.
func TerminateSession(c *gin.Context) {
	id := c.Param("id")
	if !sessionIDPattern.MatchString(id) {
		respondError(c, http.StatusNotFound, "Session not found", "")
		return
	}
	s, err := sessionManager.GetSession(c.Request.Context(), id)
	if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
		labError(c, err, "failed to load session")
		return
	}
	if err != nil || s == nil {
		respondError(c, http.StatusNotFound, "Session not found", "")
		return
	}
	if err := sessionManager.DeleteSession(c.Request.Context(), id); err != nil {
		labError(c, err, "failed to terminate session")
		return
	}
	logSelf(c, models.LevelWarn, "admin_terminate_session",
		fmt.Sprintf("Terminated session %s of user %d (task %s)", s.ID, s.UserID, s.TaskID))
	c.Status(http.StatusNoContent)
}

// GetActivityLogs handles GET /api/admin/activity?limit=1..500.
func GetActivityLogs(c *gin.Context) {
	limit, ok := parseLimit(c, 100, 1, 500)
	if !ok {
		return
	}
	out := []models.ActivityEntry{}
	if err := database.DB.WithContext(c.Request.Context()).Model(&models.ActivityLog{}).
		Select("id, username, action, details, ip_address, level, created_at").
		Order("created_at DESC, id DESC").Limit(limit).Scan(&out).Error; err != nil {
		internalError(c, err, "failed to list activity")
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetResources handles GET /api/admin/resources.
func GetResources(c *gin.Context) {
	stats, err := sessionManager.ResourceStats(c.Request.Context())
	if err != nil || stats == nil {
		if err == nil {
			err = errors.New("nil resource stats")
		}
		internalError(c, err, "failed to collect resource stats")
		return
	}
	if stats.Sessions == nil {
		stats.Sessions = []models.ResourceSessionStats{}
	}
	c.JSON(http.StatusOK, stats)
}

// ReloadTasks handles POST /api/admin/tasks/reload.
func ReloadTasks(c *gin.Context) {
	if err := task.ReloadTasks(); err != nil {
		logSelf(c, models.LevelError, "admin_reload_tasks", "Task reload failed")
		internalError(c, err, "failed to reload tasks")
		return
	}
	n := len(task.GetTasks())
	logSelf(c, models.LevelInfo, "admin_reload_tasks", fmt.Sprintf("Reloaded %d tasks", n))
	c.JSON(http.StatusOK, gin.H{"count": n})
}
