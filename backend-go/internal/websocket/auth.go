package websocket

import (
	"errors"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// MaxConnsPerUser bounds concurrent terminal sockets per user.
const MaxConnsPerUser = 3

var (
	errSessionNotFound = errors.New("session not found")
	sessionIDRe        = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

// authenticate resolves the caller; variable so tests can stub it without a database.
var authenticate = middleware.Authenticate

// findSession loads an active, unexpired session owned by userID. Variable for tests.
var findSession = func(sessionID string, userID uint) (*models.Session, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var s models.Session
	err := database.DB.
		Where("id = ? AND user_id = ? AND status = ? AND expires_at > ?", sessionID, userID, "active", time.Now()).
		Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// AuthenticateWebSocket authenticates the caller (auth cookie or Bearer header; no query-string
// tokens) and verifies the requested session is active, unexpired and owned by them.
// On failure it has already written the HTTP error response.
func AuthenticateWebSocket(c *gin.Context) (*models.Session, *middleware.Identity, error) {
	id, status, err := authenticate(c)
	if err != nil {
		switch status {
		case http.StatusForbidden:
			writeErr(c, status, "Account disabled", "This account has been deactivated")
		case http.StatusServiceUnavailable:
			writeErr(c, status, "Service unavailable", "Please try again shortly")
		default:
			writeErr(c, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		}
		return nil, nil, err
	}

	sessionID := c.Query("sessionId")
	if !sessionIDRe.MatchString(sessionID) {
		writeErr(c, http.StatusBadRequest, "Bad request", "valid sessionId query parameter is required")
		return nil, nil, errors.New("invalid session id")
	}

	sess, err := findSession(sessionID, id.UserID)
	if err != nil {
		if errors.Is(err, errSessionNotFound) {
			// Same answer for "missing" and "not yours" so ids cannot be probed.
			log.Warn().Str("session_id", sessionID).Uint("user_id", id.UserID).Msg("ws: session not found or not owned")
			writeErr(c, http.StatusNotFound, "Not found", "Session does not exist or has ended")
		} else {
			log.Error().Err(err).Msg("ws: session lookup failed")
			writeErr(c, http.StatusServiceUnavailable, "Service unavailable", "Please try again shortly")
		}
		return nil, nil, err
	}
	return sess, id, nil
}

func writeErr(c *gin.Context, status int, errMsg, message string) {
	if status == 0 || status == http.StatusOK {
		status = http.StatusUnauthorized
	}
	c.AbortWithStatusJSON(status, gin.H{"error": errMsg, "message": message, "code": status})
}

// CheckOriginFromConfig returns a gorilla CheckOrigin that accepts only origins in the CORS
// allowlist (exact match) or the request's own host. A missing Origin (non-browser client)
// is accepted because cross-site WebSocket hijacking requires a browser, which always sends it.
func CheckOriginFromConfig(allowedOrigins []string) func(r *http.Request) bool {
	allow := middleware.NewOriginAllowlist(allowedOrigins)
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if allow.OriginPermitted(origin, r) {
			return true
		}
		log.Warn().Str("origin", origin).Msg("ws: origin rejected")
		return false
	}
}

// connLimiter tracks concurrent sockets per user.
type connLimiter struct {
	mu     sync.Mutex
	counts map[uint]int
	max    int
}

func newConnLimiter(limit int) *connLimiter {
	return &connLimiter{counts: make(map[uint]int), max: limit}
}

func (l *connLimiter) acquire(userID uint) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[userID] >= l.max {
		return false
	}
	l.counts[userID]++
	return true
}

func (l *connLimiter) release(userID uint) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[userID] <= 1 {
		delete(l.counts, userID)
		return
	}
	l.counts[userID]--
}

var userConns = newConnLimiter(MaxConnsPerUser)
