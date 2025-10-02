package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `gorm:"uniqueIndex;not null" json:"username" binding:"required,min=3,max=50"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email" binding:"required,email"`
	Password  string         `gorm:"not null" json:"-"`
	Role      string         `gorm:"default:'student'" json:"role"` // student, instructor, admin
	Active    bool           `gorm:"default:true" json:"active"`
	Sessions  []Session      `gorm:"foreignKey:UserID" json:"sessions,omitempty"`
	Progress  []Progress     `gorm:"foreignKey:UserID" json:"progress,omitempty"`
}

// Session represents an active container session
type Session struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TaskID      string    `gorm:"not null;index" json:"task_id"`
	ContainerID string    `gorm:"not null" json:"container_id"`
	Status      string    `gorm:"default:'active'" json:"status"` // active, expired, stopped
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `gorm:"index" json:"expires_at"`
	LastCheckAt time.Time `json:"last_check_at,omitempty"`
}

// Progress tracks user completion and scores
type Progress struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	TaskID      string    `gorm:"not null;index" json:"task_id"`
	Score       int       `json:"score"`
	Total       int       `json:"total"`
	Passed      bool      `json:"passed"`
	Attempts    int       `gorm:"default:1" json:"attempts"`
	TimeSpent   int       `json:"time_spent"` // seconds
	CompletedAt time.Time `json:"completed_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Task represents a DevOps challenge
type Task struct {
	ID          string   `yaml:"id" json:"id"`
	Title       string   `yaml:"title" json:"title"`
	Description string   `yaml:"description" json:"description"`
	Image       string   `yaml:"image" json:"image"`
	Order       int      `yaml:"order" json:"order"`
	Difficulty  int      `yaml:"difficulty" json:"difficulty"` // 1-5
	Setup       []string `yaml:"setup" json:"setup"`
	Checks      []Check  `yaml:"checks" json:"checks"`
}

// Check represents a validation check
type Check struct {
	Name string `yaml:"name" json:"name"`
	Cmd  string `yaml:"cmd" json:"cmd"`
}

// CheckResult represents the result of a check
type CheckResult struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
}

// CheckResponse is the API response for check endpoint
type CheckResponse struct {
	Passed  bool          `json:"passed"`
	Score   int           `json:"score"`
	Total   int           `json:"total"`
	Results []CheckResult `json:"results"`
}

// LoginRequest is the login API request
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterRequest is the registration API request
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse is the login API response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// SessionCreateRequest is the session creation request
type SessionCreateRequest struct {
	TaskID string `json:"taskId" binding:"required"`
}

// SessionResponse is the session API response
type SessionResponse struct {
	SessionID  string    `json:"sessionId"`
	TTLSeconds int       `json:"ttlSeconds"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// HealthResponse is the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// StatsResponse provides platform statistics
type StatsResponse struct {
	TotalUsers     int64 `json:"total_users"`
	ActiveSessions int64 `json:"active_sessions"`
	TotalAttempts  int64 `json:"total_attempts"`
	CompletionRate float64 `json:"completion_rate"`
}

// LeaderboardEntry represents a leaderboard entry
type LeaderboardEntry struct {
	UserID    uint   `json:"user_id"`
	Username  string `json:"username"`
	Score     int    `json:"score"`
	Completed int    `json:"completed"`
	Rank      int    `json:"rank"`
}

// ActivityLog tracks user actions for audit
type ActivityLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `gorm:"not null" json:"action"` // login, logout, session_start, session_stop, task_complete, etc.
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
