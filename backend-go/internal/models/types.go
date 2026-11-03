package models

import (
	"time"

	"gorm.io/gorm"
)

// Roles a user can have.
const (
	RoleStudent = "student"
	RoleAdmin   = "admin"
)

// Session statuses.
const (
	SessionActive  = "active"
	SessionStopped = "stopped"
	SessionExpired = "expired"
)

// Activity log levels.
const (
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// User represents a user in the system.
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Username     string         `gorm:"uniqueIndex;not null" json:"username"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Password     string         `gorm:"not null" json:"-"`
	Role         string         `gorm:"default:'student'" json:"role"` // student, admin
	Active       bool           `gorm:"default:true" json:"active"`
	Github       string         `gorm:"not null;default:''" json:"github"`
	Goal         string         `gorm:"not null;default:''" json:"goal"`
	LastActiveAt *time.Time     `gorm:"index" json:"last_active_at,omitempty"`
	Sessions     []Session      `gorm:"foreignKey:UserID" json:"-"`
	Progress     []Progress     `gorm:"foreignKey:UserID" json:"-"`
}

// Session represents a lab container session.
type Session struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index:idx_sessions_user_status,priority:1" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"-"`
	TaskID      string    `gorm:"not null;index" json:"task_id"`
	ContainerID string    `gorm:"not null" json:"container_id"`
	Status      string    `gorm:"default:'active';index:idx_sessions_user_status,priority:2" json:"status"` // active, expired, stopped
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	ExpiresAt   time.Time `gorm:"index" json:"expires_at"`
	LastCheckAt time.Time `json:"last_check_at,omitempty"`
	// EndedAt is set by the session manager when the session is stopped or expires.
	EndedAt *time.Time `json:"ended_at,omitempty"`
}

// Progress tracks a user's result for one task (one row per user+task).
type Progress struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID" json:"-"`
	TaskID      string     `gorm:"not null;index" json:"task_id"`
	Score       int        `json:"score"`
	Total       int        `json:"total"`
	Passed      bool       `gorm:"not null;default:false" json:"passed"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	TimeSpent   int        `gorm:"not null;default:0" json:"time_spent"` // seconds
	CompletedAt *time.Time `gorm:"index" json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CheckAttempt records every solution check (used for streaks, achievements and admin stats).
type CheckAttempt struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index:idx_check_attempts_user_created,priority:1" json:"user_id"`
	TaskID         string    `gorm:"not null;index" json:"task_id"`
	SessionID      string    `gorm:"not null;index" json:"session_id"`
	Score          int       `json:"score"`
	Total          int       `json:"total"`
	Passed         bool      `gorm:"not null;default:false" json:"passed"`
	ElapsedSeconds int       `gorm:"not null;default:0" json:"elapsed_seconds"` // since session start
	CreatedAt      time.Time `gorm:"index:idx_check_attempts_user_created,priority:2" json:"created_at"`
}

// Task represents a DevOps challenge loaded from tasks/<id>/task.yaml.
// Setup and Checks[].Cmd reveal the solution and must never be serialized to clients.
type Task struct {
	ID          string   `yaml:"id" json:"id"`
	Title       string   `yaml:"title" json:"title"`
	Description string   `yaml:"description" json:"description"`
	Category    string   `yaml:"category" json:"category"`
	Points      int      `yaml:"points" json:"points"`
	Hints       []string `yaml:"hints" json:"hints"`
	Image       string   `yaml:"image" json:"image"`
	Order       int      `yaml:"order" json:"order"`
	Difficulty  int      `yaml:"difficulty" json:"difficulty"` // 1-3
	Setup       []string `yaml:"setup" json:"-"`
	Checks      []Check  `yaml:"checks" json:"checks"`
	// Readme is the markdown from README.md next to task.yaml.
	Readme string `yaml:"-" json:"readme"`
}

// Check represents a validation check.
type Check struct {
	Name string `yaml:"name" json:"name"`
	Cmd  string `yaml:"cmd" json:"-"`
}

// CheckResult represents the result of a check.
type CheckResult struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
}

// CheckResponse is the API response for the check endpoint.
type CheckResponse struct {
	Passed  bool          `json:"passed"`
	Score   int           `json:"score"`
	Total   int           `json:"total"`
	Results []CheckResult `json:"results"`
}

// LoginRequest is the login API request.
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=72"`
}

// RegisterRequest is the registration API request.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// UpdateMeRequest is the PUT /api/me request; nil fields are left unchanged.
type UpdateMeRequest struct {
	Email  *string `json:"email" binding:"omitempty,email,max=254"`
	Github *string `json:"github" binding:"omitempty,max=39"`
	Goal   *string `json:"goal" binding:"omitempty,max=800"` // rune length (≤200) is checked explicitly
}

// ChangePasswordRequest is the POST /api/me/password request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,max=72"`
	NewPassword     string `json:"new_password" binding:"required,max=72"`
}

// AdminUpdateUserRequest is the PUT /api/admin/users/:id request.
type AdminUpdateUserRequest struct {
	Role   *string `json:"role" binding:"omitempty,oneof=student admin"`
	Active *bool   `json:"active"`
}

// UserResponse is the public representation of a user.
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	Github    string    `json:"github"`
	Goal      string    `json:"goal"`
	CreatedAt time.Time `json:"created_at"`
}

// NewUserResponse converts a User into its API representation.
func NewUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Active:    u.Active,
		Github:    u.Github,
		Goal:      u.Goal,
		CreatedAt: u.CreatedAt,
	}
}

// AuthResponse is the body of login/register responses (the JWT is only set as a cookie).
type AuthResponse struct {
	User UserResponse `json:"user"`
}

// AdminUser is an element of GET /api/admin/users.
type AdminUser struct {
	UserResponse
	Score      int        `json:"score"`
	Completed  int        `json:"completed"`
	LastActive *time.Time `json:"last_active"`
}

// SessionCreateRequest is the session creation request.
type SessionCreateRequest struct {
	TaskID string `json:"taskId" binding:"required,max=64"`
}

// SessionResponse is the session API response.
type SessionResponse struct {
	SessionID  string    `json:"sessionId"`
	TaskID     string    `json:"taskId"`
	TTLSeconds int       `json:"ttlSeconds"`
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// HealthResponse is the health check response.
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// TaskSummary is an element of GET /api/tasks.
type TaskSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  int    `json:"difficulty"`
	Points      int    `json:"points"`
	Order       int    `json:"order"`
	ChecksCount int    `json:"checks_count"`
	Unlocked    bool   `json:"unlocked"`
	Completed   bool   `json:"completed"`
	InProgress  bool   `json:"in_progress"`
}

// CheckName is the client-safe view of a check (no command).
type CheckName struct {
	Name string `json:"name"`
}

// TaskDetail is the response of GET /api/tasks/:id.
type TaskDetail struct {
	TaskSummary
	Image  string      `json:"image"`
	Readme string      `json:"readme"`
	Hints  []string    `json:"hints"`
	Checks []CheckName `json:"checks"`
}

// ProgressItem is one row of ProgressResponse.progress.
type ProgressItem struct {
	TaskID      string     `json:"task_id"`
	Score       int        `json:"score"`
	Total       int        `json:"total"`
	Passed      bool       `json:"passed"`
	Attempts    int        `json:"attempts"`
	TimeSpent   int        `json:"time_spent"`
	CompletedAt *time.Time `json:"completed_at"`
}

// SkillProgress is the completion percentage of one task category.
type SkillProgress struct {
	Category string `json:"category"`
	Percent  int    `json:"percent"`
}

// Achievement is a computed badge (there is no achievements table).
type Achievement struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Earned      bool       `json:"earned"`
	EarnedAt    *time.Time `json:"earned_at,omitempty"`
}

// ProgressResponse is the response of GET /api/progress.
type ProgressResponse struct {
	Progress         []ProgressItem  `json:"progress"`
	TotalScore       int             `json:"totalScore"`
	TotalPossible    int             `json:"totalPossible"`
	Percentage       int             `json:"percentage"`
	Completed        int             `json:"completed"`
	TotalTasks       int             `json:"totalTasks"`
	TimeInLabSeconds int64           `json:"timeInLabSeconds"`
	StreakDays       int             `json:"streakDays"`
	Skills           []SkillProgress `json:"skills"`
	Achievements     []Achievement   `json:"achievements"`
}

// LeaderboardEntry represents a leaderboard entry.
type LeaderboardEntry struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Score     int    `json:"score"`
	Completed int    `json:"completed"`
	Rank      int    `json:"rank"`
}

// LeaderboardResponse is the response of GET /api/leaderboard.
type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
	Me      *LeaderboardEntry  `json:"me"`
}

// TaskCompletion is the per-task block of the admin stats.
type TaskCompletion struct {
	TaskID   string `json:"task_id"`
	Attempts int64  `json:"attempts"`
	Passed   int64  `json:"passed"`
}

// StatsResponse provides platform statistics (GET /api/admin/stats).
type StatsResponse struct {
	TotalUsers     int64            `json:"total_users"`
	ActiveUsers    int64            `json:"active_users"`
	ActiveSessions int64            `json:"active_sessions"`
	TotalAttempts  int64            `json:"total_attempts"`
	CompletionRate float64          `json:"completion_rate"`
	Sessions24h    []int64          `json:"sessions_24h"`
	TaskCompletion []TaskCompletion `json:"task_completion"`
}

// AdminSession is an element of GET /api/admin/sessions.
type AdminSession struct {
	ID        string    `json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ActivityLog tracks user actions for audit.
type ActivityLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `gorm:"not null" json:"action"` // login, logout, session_start, session_stop, task_complete, ...
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	Level     string    `gorm:"size:8;not null;default:'INFO'" json:"level"` // INFO, WARN, ERROR
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// ActivityEntry is an element of GET /api/admin/activity.
type ActivityEntry struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	Level     string    `json:"level"`
	CreatedAt time.Time `json:"created_at"`
}

// ResourceSessionStats is the per-session container usage reported by GET /api/admin/resources.
type ResourceSessionStats struct {
	SessionID     string  `json:"session_id"`
	Username      string  `json:"username"`
	TaskID        string  `json:"task_id"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemBytes      uint64  `json:"mem_bytes"`
	MemLimitBytes uint64  `json:"mem_limit_bytes"`
}

// ResourceStats is the response of GET /api/admin/resources.
type ResourceStats struct {
	ContainersRunning int                    `json:"containers_running"`
	ContainersLimit   int                    `json:"containers_limit"`
	Sessions          []ResourceSessionStats `json:"sessions"`
}
