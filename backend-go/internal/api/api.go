// Package api implements the HTTP handlers described in docs/API.md.
package api

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LabManager is the subset of *session.Manager used by the handlers.
type LabManager interface {
	CreateSession(ctx context.Context, userID uint, taskID string) (*models.SessionResponse, error)
	GetSession(ctx context.Context, sessionID string) (*models.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	CheckSolution(ctx context.Context, sessionID string) (*models.CheckResponse, error)
	Recreate(ctx context.Context, sessionID string) (*models.SessionResponse, error)
	ActiveSessionForUser(ctx context.Context, userID uint) (*models.Session, error)
	ResourceStats(ctx context.Context) (*models.ResourceStats, error)
	// TaskAvailability reports whether this server can run the task, and why not.
	TaskAvailability(taskID string) (available bool, reason string)
}

// Options are the process-wide settings the handlers need besides the lab manager.
type Options struct {
	// Version is the build version reported by /healthz (main.version, set via -ldflags).
	Version string
	// DefaultImage is the lab image used for tasks that do not name their own image.
	DefaultImage string
}

var (
	sessionManager LabManager
	startTime      = time.Now()
	appVersion     = "dev"
	defaultImage   string
)

// Init wires the lab session manager and process settings into the handlers.
func Init(sm LabManager, opts Options) {
	sessionManager = sm
	startTime = time.Now()
	appVersion = opts.Version
	if appVersion == "" {
		appVersion = "dev"
	}
	defaultImage = opts.DefaultImage
}

var (
	taskIDPattern    = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)
	sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	githubPattern    = regexp.MustCompile(`^[A-Za-z0-9-]{1,39}$`)
)

// maxGoalRunes is the maximum length of the profile "goal" field.
const maxGoalRunes = 200

// respondError writes the standard {error, message?, code} body.
func respondError(c *gin.Context, status int, errMsg, message string) {
	c.AbortWithStatusJSON(status, models.ErrorResponse{Error: errMsg, Message: message, Code: status})
}

// internalError logs err with context and returns a generic 500.
func internalError(c *gin.Context, err error, msg string) {
	log.Error().Err(err).Str("path", c.FullPath()).Str("request_id", c.GetString("request_id")).Msg(msg)
	respondError(c, http.StatusInternalServerError, "Internal server error", "")
}

// bindJSON binds the request body; on failure it responds 413 (body too large) or 400 with msg.
func bindJSON(c *gin.Context, dst any, msg string) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		if middleware.IsBodyTooLarge(err) {
			respondError(c, http.StatusRequestEntityTooLarge, "Request body too large", "")
			return false
		}
		respondError(c, http.StatusBadRequest, "Invalid request", msg)
		return false
	}
	return true
}

// currentUser returns the authenticated identity set by the auth middleware.
func currentUser(c *gin.Context) (id uint, username, role string, ok bool) {
	id = c.GetUint("user_id")
	username = c.GetString("username")
	role = c.GetString("role")
	return id, username, role, id != 0
}

func isAdmin(c *gin.Context) bool {
	return c.GetString("role") == models.RoleAdmin
}

// parseLimit parses an optional ?limit query parameter, validating lo..hi.
func parseLimit(c *gin.Context, def, lo, hi int) (int, bool) {
	raw := c.Query("limit")
	if raw == "" {
		return def, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		respondError(c, http.StatusBadRequest, "Invalid limit", "limit must be between "+strconv.Itoa(lo)+" and "+strconv.Itoa(hi))
		return 0, false
	}
	return n, true
}

// parseOffset parses an optional non-negative ?offset query parameter.
func parseOffset(c *gin.Context) (int, bool) {
	raw := c.Query("offset")
	if raw == "" {
		return 0, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 || n > 1_000_000 {
		respondError(c, http.StatusBadRequest, "Invalid offset", "")
		return 0, false
	}
	return n, true
}

// truncate shortens s to at most n runes.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

// logActivity writes an audit entry. Failures are logged, never returned to the client.
func logActivity(c *gin.Context, userID uint, username, level, action, details string) {
	if database.DB == nil {
		return
	}
	entry := models.ActivityLog{
		UserID:    userID,
		Username:  truncate(username, 64),
		Action:    action,
		Details:   truncate(details, 512),
		IPAddress: truncate(c.ClientIP(), 64),
		UserAgent: truncate(c.GetHeader("User-Agent"), 256),
		Level:     level,
	}
	if err := database.DB.WithContext(context.WithoutCancel(c.Request.Context())).Create(&entry).Error; err != nil {
		log.Error().Err(err).Str("action", action).Msg("failed to write activity log")
	}
}

// logSelf writes an audit entry for the authenticated caller.
func logSelf(c *gin.Context, level, action, details string) {
	id, username, _, _ := currentUser(c)
	logActivity(c, id, username, level, action, details)
}

// lastActive throttles last_active_at writes to one per user per interval.
var lastActive = struct {
	sync.Mutex
	m map[uint]time.Time
}{m: map[uint]time.Time{}}

const lastActiveInterval = 2 * time.Minute

// touchLastActive records the caller's last activity time (at most once per lastActiveInterval).
func touchLastActive() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		id := c.GetUint("user_id")
		if id == 0 || database.DB == nil {
			return
		}
		now := time.Now().UTC()
		lastActive.Lock()
		prev, seen := lastActive.m[id]
		if seen && now.Sub(prev) < lastActiveInterval {
			lastActive.Unlock()
			return
		}
		if len(lastActive.m) > 50000 {
			lastActive.m = map[uint]time.Time{}
		}
		lastActive.m[id] = now
		lastActive.Unlock()
		markActive(c.Request.Context(), id, now)
	}
}

func markActive(ctx context.Context, userID uint, now time.Time) {
	if err := database.DB.WithContext(context.WithoutCancel(ctx)).Model(&models.User{}).
		Where("id = ?", userID).UpdateColumn("last_active_at", now).Error; err != nil {
		log.Warn().Err(err).Uint("user_id", userID).Msg("failed to update last_active_at")
	}
}
