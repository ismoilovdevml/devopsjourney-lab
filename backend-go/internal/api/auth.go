package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const msgInvalidCredentials = "invalid username or password"

// dummyHash is compared against when the user does not exist so that login timing
// does not reveal whether a username is registered.
var (
	dummyHashOnce sync.Once
	dummyHash     string
)

func getDummyHash() string {
	dummyHashOnce.Do(func() {
		h, err := auth.HashPassword("dummy-password-for-timing")
		if err != nil {
			log.Error().Err(err).Msg("failed to compute dummy hash")
		}
		dummyHash = h
	})
	return dummyHash
}

// issueSession sets the auth cookie for user. Returns false (and responds) on failure.
func issueSession(c *gin.Context, user *models.User) bool {
	token, err := auth.GenerateToken(user)
	if err != nil {
		internalError(c, err, "failed to generate token")
		return false
	}
	auth.SetAuthCookie(c, token)
	return true
}

// Register handles POST /api/auth/register.
func Register(c *gin.Context) {
	var req models.RegisterRequest
	if !bindJSON(c, &req, "username, email and password are required") {
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if err := auth.ValidateUsername(req.Username); err != nil {
		respondError(c, http.StatusBadRequest, "Invalid username", err.Error())
		return
	}
	if err := auth.ValidatePasswordStrength(req.Password, req.Username); err != nil {
		respondError(c, http.StatusBadRequest, "Weak password", err.Error())
		return
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		internalError(c, err, "failed to hash password")
		return
	}

	user := models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
		Role:     models.RoleStudent,
		Active:   true,
	}

	var exists int64
	if err := database.DB.WithContext(c.Request.Context()).Unscoped().Model(&models.User{}).
		Where("username = ? OR email = ?", req.Username, req.Email).Count(&exists).Error; err != nil {
		internalError(c, err, "failed to check existing user")
		return
	}
	if exists > 0 {
		respondError(c, http.StatusConflict, "User already exists", "username or email is already registered")
		return
	}
	// The unique indexes are the real guard against concurrent registrations.
	if err := database.DB.WithContext(c.Request.Context()).Create(&user).Error; err != nil {
		if database.IsUniqueViolation(err) {
			respondError(c, http.StatusConflict, "User already exists", "username or email is already registered")
			return
		}
		internalError(c, err, "failed to create user")
		return
	}

	if !issueSession(c, &user) {
		return
	}
	logActivity(c, user.ID, user.Username, models.LevelInfo, "register", "New user registered")
	c.JSON(http.StatusCreated, models.AuthResponse{User: models.NewUserResponse(&user)})
}

// Login handles POST /api/auth/login.
func Login(c *gin.Context) {
	var req models.LoginRequest
	if !bindJSON(c, &req, "username and password are required") {
		return
	}

	var user models.User
	err := database.DB.WithContext(c.Request.Context()).Where("username = ?", req.Username).Take(&user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		internalError(c, err, "failed to load user")
		return
	}
	found := err == nil

	// Always run bcrypt so response time does not reveal whether the user exists.
	hash := getDummyHash()
	if found {
		hash = user.Password
	}
	ok := auth.CheckPassword(req.Password, hash) && found

	if !ok {
		var uid uint
		if found {
			uid = user.ID
		}
		logActivity(c, uid, truncate(req.Username, 64), models.LevelWarn, "login_failed", "Invalid username or password")
		respondError(c, http.StatusUnauthorized, "Unauthorized", msgInvalidCredentials)
		return
	}

	if !user.Active {
		logActivity(c, user.ID, user.Username, models.LevelWarn, "login_failed", "Login attempt on deactivated account")
		respondError(c, http.StatusForbidden, "Account disabled", "This account has been deactivated")
		return
	}

	if !issueSession(c, &user) {
		return
	}
	markActive(c.Request.Context(), user.ID, nowUTC())
	logActivity(c, user.ID, user.Username, models.LevelInfo, "login", "User logged in")
	c.JSON(http.StatusOK, models.AuthResponse{User: models.NewUserResponse(&user)})
}

// Logout handles POST /api/auth/logout.
func Logout(c *gin.Context) {
	auth.ClearAuthCookie(c)
	logSelf(c, models.LevelInfo, "logout", "User logged out")
	c.Status(http.StatusNoContent)
}

// loadCurrentUser loads the caller's row; responds 401 when it no longer exists.
func loadCurrentUser(c *gin.Context) (*models.User, bool) {
	id, _, _, ok := currentUser(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "Unauthorized", "Authentication required")
		return nil, false
	}
	var user models.User
	if err := database.DB.WithContext(c.Request.Context()).Take(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "Unauthorized", "Authentication required")
			return nil, false
		}
		internalError(c, err, "failed to load user")
		return nil, false
	}
	return &user, true
}

// GetMe handles GET /api/me.
func GetMe(c *gin.Context) {
	user, ok := loadCurrentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, models.NewUserResponse(user))
}

// validateProfile checks the optional PUT /api/me fields and returns the column updates.
func validateProfile(req *models.UpdateMeRequest) (map[string]any, string) {
	updates := map[string]any{}
	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email == "" {
			return nil, "email must not be empty"
		}
		updates["email"] = email
	}
	if req.Github != nil {
		gh := strings.TrimSpace(*req.Github)
		if gh != "" && !githubPattern.MatchString(gh) {
			return nil, "github must be a GitHub username (letters, digits, '-', max 39)"
		}
		updates["github"] = gh
	}
	if req.Goal != nil {
		goal := strings.TrimSpace(*req.Goal)
		if utf8.RuneCountInString(goal) > maxGoalRunes {
			return nil, "goal must be at most 200 characters"
		}
		if strings.ContainsAny(goal, "\x00") {
			return nil, "goal contains invalid characters"
		}
		updates["goal"] = goal
	}
	return updates, ""
}

// UpdateMe handles PUT /api/me.
func UpdateMe(c *gin.Context) {
	var req models.UpdateMeRequest
	if !bindJSON(c, &req, "email must be valid; github max 39 characters; goal max 200 characters") {
		return
	}
	updates, msg := validateProfile(&req)
	if msg != "" {
		respondError(c, http.StatusBadRequest, "Invalid request", msg)
		return
	}

	user, ok := loadCurrentUser(c)
	if !ok {
		return
	}
	if len(updates) > 0 {
		if err := database.DB.WithContext(c.Request.Context()).Model(user).Updates(updates).Error; err != nil {
			if database.IsUniqueViolation(err) {
				respondError(c, http.StatusConflict, "Email already in use", "")
				return
			}
			internalError(c, err, "failed to update profile")
			return
		}
		if v, ok := updates["email"].(string); ok {
			user.Email = v
		}
		if v, ok := updates["github"].(string); ok {
			user.Github = v
		}
		if v, ok := updates["goal"].(string); ok {
			user.Goal = v
		}
		logSelf(c, models.LevelInfo, "profile_update", "Profile updated")
	}
	c.JSON(http.StatusOK, models.NewUserResponse(user))
}

// ChangePassword handles POST /api/me/password.
func ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if !bindJSON(c, &req, "current_password and new_password are required") {
		return
	}
	user, ok := loadCurrentUser(c)
	if !ok {
		return
	}
	if !auth.CheckPassword(req.CurrentPassword, user.Password) {
		logSelf(c, models.LevelWarn, "password_change_failed", "Wrong current password")
		respondError(c, http.StatusUnauthorized, "Unauthorized", "current password is incorrect")
		return
	}
	if err := auth.ValidatePasswordStrength(req.NewPassword, user.Username); err != nil {
		respondError(c, http.StatusBadRequest, "Weak password", err.Error())
		return
	}
	if req.NewPassword == req.CurrentPassword {
		respondError(c, http.StatusBadRequest, "Weak password", "new password must differ from the current one")
		return
	}
	hashed, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		internalError(c, err, "failed to hash password")
		return
	}
	// Every token issued before this instant is rejected by middleware.Authenticate.
	changedAt := nowUTC().Truncate(time.Microsecond) // Postgres precision: stored == compared
	if err := database.DB.WithContext(c.Request.Context()).Model(user).UpdateColumns(map[string]any{
		"password":            hashed,
		"password_changed_at": changedAt,
	}).Error; err != nil {
		internalError(c, err, "failed to update password")
		return
	}
	user.Password = hashed
	user.PasswordChangedAt = &changedAt
	middleware.InvalidateUserCache(user.ID)
	logSelf(c, models.LevelInfo, "password_change", "Password changed")
	// Other devices are signed out; keep this client signed in with a fresh token.
	if !issueSession(c, user) {
		return
	}
	c.Status(http.StatusNoContent)
}
