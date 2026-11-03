package middleware

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Identity is the authenticated principal resolved from a request.
type Identity struct {
	UserID   uint
	Username string
	Role     string
}

// UserState is the subset of the user row needed to authorize a request.
type UserState struct {
	ID       uint
	Username string
	Role     string
	Active   bool
}

var (
	// ErrUnauthenticated means no valid credentials were presented.
	ErrUnauthenticated = errors.New("authentication required")
	// ErrUserDisabled means the token is valid but the user is deleted or deactivated.
	ErrUserDisabled = errors.New("account disabled")
	// errUserNotFound is returned by lookupUser when the row is missing or soft-deleted.
	errUserNotFound = errors.New("user not found")
)

// lookupUser loads the user's current state. It is a variable so tests can stub it without Postgres.
var lookupUser = func(id uint) (*UserState, error) {
	if database.DB == nil {
		return nil, errors.New("database not initialized")
	}
	var u models.User
	err := database.DB.Select("id", "username", "role", "active").Where("id = ?", id).Take(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UserState{ID: u.ID, Username: u.Username, Role: u.Role, Active: u.Active}, nil
}

// userCacheTTL bounds how long a deactivation/role change can take to apply.
const userCacheTTL = 30 * time.Second

type cachedUser struct {
	state   *UserState // nil means "not found / deleted"
	expires time.Time
}

var userCache = struct {
	sync.Mutex
	m map[uint]cachedUser
}{m: make(map[uint]cachedUser)}

// InvalidateUserCache drops the cached state for a user. Call it after changing a user's
// role/active flag or deleting them so the change applies immediately on this instance.
func InvalidateUserCache(userID uint) {
	userCache.Lock()
	delete(userCache.m, userID)
	userCache.Unlock()
}

func resetUserCache() {
	userCache.Lock()
	userCache.m = make(map[uint]cachedUser)
	userCache.Unlock()
}

func getUserState(id uint) (*UserState, error) {
	now := time.Now()
	userCache.Lock()
	if e, ok := userCache.m[id]; ok && now.Before(e.expires) {
		userCache.Unlock()
		if e.state == nil {
			return nil, errUserNotFound
		}
		return e.state, nil
	}
	userCache.Unlock()

	state, err := lookupUser(id)
	if err != nil && !errors.Is(err, errUserNotFound) {
		return nil, err // transient DB error: do not cache
	}

	userCache.Lock()
	if len(userCache.m) > 10000 { // crude bound against unbounded growth
		userCache.m = make(map[uint]cachedUser)
	}
	userCache.m[id] = cachedUser{state: state, expires: now.Add(userCacheTTL)}
	userCache.Unlock()

	if state == nil {
		return nil, errUserNotFound
	}
	return state, nil
}

// Authenticate resolves the caller from the auth cookie or Bearer header, validates the JWT
// and checks the user still exists and is active. On failure it returns the HTTP status to use.
// Role and username come from the database (not the token) so demotions apply within userCacheTTL.
func Authenticate(c *gin.Context) (*Identity, int, error) {
	token := auth.TokenFromRequest(c)
	if token == "" {
		return nil, http.StatusUnauthorized, ErrUnauthenticated
	}
	claims, err := auth.ValidateToken(token)
	if err != nil {
		return nil, http.StatusUnauthorized, ErrUnauthenticated
	}
	state, err := getUserState(claims.UserID)
	if err != nil {
		if errors.Is(err, errUserNotFound) {
			return nil, http.StatusUnauthorized, ErrUnauthenticated
		}
		log.Error().Err(err).Uint("user_id", claims.UserID).Msg("auth: user lookup failed")
		return nil, http.StatusServiceUnavailable, err
	}
	if !state.Active {
		return nil, http.StatusForbidden, ErrUserDisabled
	}
	return &Identity{UserID: state.ID, Username: state.Username, Role: state.Role}, http.StatusOK, nil
}

func setIdentity(c *gin.Context, id *Identity) {
	c.Set("user_id", id.UserID)
	c.Set("username", id.Username)
	c.Set("role", id.Role)
}

// AuthMiddleware requires a valid, active user and sets "user_id" (uint), "username" and "role".
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, status, err := Authenticate(c)
		if err != nil {
			switch status {
			case http.StatusForbidden:
				abortJSON(c, status, "Account disabled", "This account has been deactivated")
			case http.StatusServiceUnavailable:
				abortJSON(c, status, "Service unavailable", "Please try again shortly")
			default:
				abortJSON(c, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			}
			return
		}
		setIdentity(c, id)
		c.Next()
	}
}

// OptionalAuthMiddleware sets the identity when valid credentials are present; it never aborts.
func OptionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if id, _, err := Authenticate(c); err == nil {
			setIdentity(c, id)
		}
		c.Next()
	}
}

// RoleMiddleware allows the request only if the authenticated role is one of allowedRoles.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		userRole, _ := role.(string)
		if userRole != "" {
			for _, allowed := range allowedRoles {
				if userRole == allowed {
					c.Next()
					return
				}
			}
		}
		abortJSON(c, http.StatusForbidden, "Forbidden", "Insufficient permissions")
	}
}

// abortJSON writes the standard error body {error, message, code} and aborts.
func abortJSON(c *gin.Context, status int, errMsg, message string) {
	body := gin.H{"error": errMsg, "code": status}
	if message != "" {
		body["message"] = message
	}
	c.AbortWithStatusJSON(status, body)
}
