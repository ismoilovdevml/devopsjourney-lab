package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var sessionManager *session.Manager
var startTime time.Time

// Init initializes API handlers
func Init(sm *session.Manager) {
	sessionManager = sm
	startTime = time.Now()
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the API is running
// @Tags health
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Router /healthz [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "2.0.0",
		Uptime:    time.Since(startTime).String(),
	})
}

// Register godoc
// @Summary Register new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} models.LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Router /api/auth/register [post]
func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Interface("headers", c.Request.Header).Msg("Failed to bind registration JSON")

		// User-friendly error messages
		errorMsg := "Invalid request format"
		if strings.Contains(err.Error(), "Password") && strings.Contains(err.Error(), "min") {
			errorMsg = "Password must be at least 8 characters long"
		} else if strings.Contains(err.Error(), "Email") {
			errorMsg = "Please provide a valid email address"
		} else if strings.Contains(err.Error(), "Username") && strings.Contains(err.Error(), "min") {
			errorMsg = "Username must be at least 3 characters long"
		}

		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   errorMsg,
			Message: errorMsg,
			Code:    http.StatusBadRequest,
		})
		return
	}

	log.Info().Str("username", req.Username).Str("email", req.Email).Msg("Registration request received")

	// Hash password before transaction
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Error().Err(err).Msg("Failed to hash password")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create user",
			Code:  http.StatusInternalServerError,
		})
		return
	}

	// Use transaction to prevent race condition
	var user models.User
	err = database.WithTransaction(c.Request.Context(), func(tx *gorm.DB) error {
		// Check if user exists within transaction (locks row)
		var existingUser models.User
		err := tx.Where("username = ? OR email = ?", req.Username, req.Email).
			First(&existingUser).Error

		if err == nil {
			return fmt.Errorf("username or email already exists")
		}

		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("database error: %w", err)
		}

		// Create user within transaction
		user = models.User{
			Username: req.Username,
			Email:    req.Email,
			Password: hashedPassword,
			Role:     "student",
			Active:   true,
		}

		return tx.Create(&user).Error
	})

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "User already exists",
				Message: "Username or email is already registered",
				Code:    http.StatusConflict,
			})
		} else {
			log.Error().Err(err).Msg("Failed to create user")
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "Failed to create user",
				Code:  http.StatusInternalServerError,
			})
		}
		return
	}

	// Generate token
	token, err := auth.GenerateToken(&user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate token",
			Code: http.StatusInternalServerError,
		})
		return
	}

	log.Info().Str("username", user.Username).Msg("User registered")

	c.JSON(http.StatusCreated, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// Login godoc
// @Summary Login
// @Description Authenticate user and get JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/auth/login [post]
func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().Err(err).Interface("headers", c.Request.Header).Msg("Failed to bind login JSON")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request",
			Message: err.Error(),
			Code: http.StatusBadRequest,
		})
		return
	}

	log.Info().Str("username", req.Username).Msg("Login request received")

	// Find user
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "User not found",
			Message: "The username you entered does not exist. Please register first.",
			Code: http.StatusUnauthorized,
		})
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error: "Invalid password",
			Message: "The password you entered is incorrect. Please try again.",
			Code: http.StatusUnauthorized,
		})
		return
	}

	// Check if active
	if !user.Active {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Error: "Account is deactivated",
			Code: http.StatusForbidden,
		})
		return
	}

	// Generate token
	token, err := auth.GenerateToken(&user)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate token")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to generate token",
			Code: http.StatusInternalServerError,
		})
		return
	}

	log.Info().Str("username", user.Username).Msg("User logged in")

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// GetTasks godoc
// @Summary List all tasks
// @Description Get list of all available tasks with unlock status
// @Tags tasks
// @Produce json
// @Success 200 {array} models.Task
// @Router /api/tasks [get]
func GetTasks(c *gin.Context) {
	tasks := task.GetTasks()

	// Get user progress if authenticated - Single query optimization
	userID, exists := c.Get("user_id")
	completedMap := make(map[string]bool)

	if exists {
		var progress []models.Progress
		// Optimized: single query with only task_id
		database.DB.Where("user_id = ? AND passed = ?", userID, true).
			Select("task_id").
			Find(&progress)

		for _, p := range progress {
			completedMap[p.TaskID] = true
		}
	}

	// Return tasks with unlock status - O(n) complexity instead of O(n²)
	simplified := make([]gin.H, 0, len(tasks))
	prevTaskCompleted := true // First task is always unlocked

	for i, t := range tasks {
		// Task is unlocked if it's first OR previous task is completed
		unlocked := i == 0 || prevTaskCompleted
		completed := completedMap[t.ID]

		simplified = append(simplified, gin.H{
			"id":          t.ID,
			"title":       t.Title,
			"image":       t.Image,
			"order":       t.Order,
			"difficulty":  t.Difficulty,
			"unlocked":    unlocked,
			"completed":   completed,
			"description": t.Description,
			})

		// Update for next iteration
		prevTaskCompleted = completed
	}

	c.JSON(http.StatusOK, simplified)
}

// GetTask godoc
// @Summary Get task details
// @Description Get detailed information about a specific task
// @Tags tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} models.Task
// @Failure 404 {object} models.ErrorResponse
// @Router /api/tasks/{id} [get]
func GetTask(c *gin.Context) {
	taskID := c.Param("id")

	taskObj, err := task.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Task not found",
			Code: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, taskObj)
}

// CreateSession godoc
// @Summary Create new session
// @Description Create a new container session for a task
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body models.SessionCreateRequest true "Session details"
// @Success 201 {object} models.SessionResponse
// @Failure 400,401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/sessions [post]
func CreateSession(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req models.SessionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Invalid request",
			Message: err.Error(),
			Code: http.StatusBadRequest,
		})
		return
	}

	session, err := sessionManager.CreateSession(c.Request.Context(), userID, req.TaskID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create session")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to create session",
			Message: err.Error(),
			Code: http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetSession godoc
// @Summary Get session info
// @Description Get information about a session
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.Session
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/sessions/{id} [get]
func GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	session, err := sessionManager.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Session not found",
			Code: http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession godoc
// @Summary Delete session
// @Description Delete a session and cleanup container
// @Tags sessions
// @Param id path string true "Session ID"
// @Success 204
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/sessions/{id} [delete]
func DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")

	if err := sessionManager.DeleteSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Session not found",
			Code: http.StatusNotFound,
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// CheckSolution godoc
// @Summary Check solution
// @Description Run validation checks on a session
// @Tags sessions
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.CheckResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /api/check/{id} [post]
func CheckSolution(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetUint("user_id")

	result, err := sessionManager.CheckSolution(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error: "Failed to check solution",
			Message: err.Error(),
			Code: http.StatusNotFound,
		})
		return
	}

	// Save progress if passed
	if result.Passed {
		session, _ := sessionManager.GetSession(c.Request.Context(), sessionID)
		if session != nil {
			progress := models.Progress{
				UserID:      userID,
				TaskID:      session.TaskID,
				Score:       result.Score,
				Total:       result.Total,
				Passed:      true,
				CompletedAt: time.Now(),
			}
			database.DB.Create(&progress)
		}
	}

	c.JSON(http.StatusOK, result)
}
