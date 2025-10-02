package api

import (
	"fmt"
	"net/http"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"github.com/gin-gonic/gin"
)

// GetUserProgress returns user's progress
func GetUserProgress(c *gin.Context) {
	userID := c.GetUint("user_id")

	var progress []models.Progress
	if err := database.DB.Where("user_id = ?", userID).Order("completed_at DESC").Find(&progress).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch progress"})
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetLeaderboard returns top users by score
func GetLeaderboard(c *gin.Context) {
	var leaderboard []models.LeaderboardEntry

	query := `
		SELECT
			u.id as user_id,
			u.username,
			SUM(p.score) as score,
			COUNT(DISTINCT p.task_id) as completed,
			ROW_NUMBER() OVER (ORDER BY SUM(p.score) DESC) as rank
		FROM users u
		LEFT JOIN progresses p ON u.id = p.user_id AND p.passed = true
		WHERE u.active = true
		GROUP BY u.id, u.username
		ORDER BY score DESC
		LIMIT 100
	`

	if err := database.DB.Raw(query).Scan(&leaderboard).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	c.JSON(http.StatusOK, leaderboard)
}

// GetStats returns platform statistics (admin only)
func GetStats(c *gin.Context) {
	var totalUsers, onlineUsers, activeSessions, completedTasks int64

	// Total users
	database.DB.Model(&models.User{}).Count(&totalUsers)

	// Online users (active in last 10 minutes - we'll implement this later)
	onlineUsers = 0

	// Active sessions
	database.DB.Model(&models.Session{}).Where("status = ?", "active").Count(&activeSessions)

	// Completed tasks
	database.DB.Model(&models.Progress{}).Where("passed = ?", true).Count(&completedTasks)

	c.JSON(http.StatusOK, gin.H{
		"users": gin.H{
			"total":   totalUsers,
			"online":  onlineUsers,
			"offline": totalUsers - onlineUsers,
		},
		"sessions": gin.H{
			"active": activeSessions,
		},
		"progress": gin.H{
			"completedTasks": completedTasks,
			"totalScore":     0,
		},
		"activity24h": []interface{}{},
	})
}

// GetUsers returns all users with stats (admin only)
func GetUsers(c *gin.Context) {
	var users []models.User

	if err := database.DB.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Enhance with additional stats
	result := make([]gin.H, len(users))
	for i, user := range users {
		var sessionCount, totalScore int64
		database.DB.Model(&models.Session{}).Where("user_id = ?", user.ID).Count(&sessionCount)
		database.DB.Model(&models.Progress{}).Where("user_id = ? AND passed = ?", user.ID, true).
			Select("COALESCE(SUM(score), 0)").Row().Scan(&totalScore)

		result[i] = gin.H{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"role":          user.Role,
			"active":        user.Active,
			"created_at":    user.CreatedAt,
			"is_online":     false, // TODO: implement online status
			"session_count": sessionCount,
			"total_score":   totalScore,
			"last_login":    user.CreatedAt, // TODO: implement last_login tracking
		}
	}

	c.JSON(http.StatusOK, result)
}

// UpdateUser updates user details (admin only)
func UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		Active   *bool  `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	updates := make(map[string]interface{})

	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.Password != "" {
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updates["password"] = hashedPassword
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Log activity
	adminID := c.GetUint("user_id")
	adminUsername, _ := c.Get("username")
	activityLog := models.ActivityLog{
		UserID:    adminID,
		Username:  adminUsername.(string),
		Action:    "admin_update_user",
		Details:   "Admin updated user ID: " + userID,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	database.DB.Create(&activityLog)

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// ReloadTasks reloads all tasks (admin only)
func ReloadTasks(c *gin.Context) {
	if err := task.ReloadTasks(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload tasks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tasks reloaded successfully",
		"count":   len(task.GetTasks()),
	})
}

// GetUserDetails returns detailed user information (admin only)
func GetUserDetails(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get sessions
	var sessions []models.Session
	database.DB.Where("user_id = ?", user.ID).Find(&sessions)

	// Get progress
	var progress []models.Progress
	database.DB.Where("user_id = ?", user.ID).Order("completed_at DESC").Find(&progress)

	// Prepare user data
	userData := gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"active":     user.Active,
		"created_at": user.CreatedAt,
		"is_online":  false,
		"last_login": user.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     userData,
		"sessions": sessions,
		"progress": progress,
		"activity": []interface{}{}, // TODO: implement activity logs
	})
}

// GetAllSessions returns all active sessions (admin only)
func GetAllSessions(c *gin.Context) {
	var sessions []models.Session
	if err := database.DB.Where("status = ?", "active").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
		return
	}

	// Enhance with usernames
	result := make([]gin.H, len(sessions))
	for i, session := range sessions {
		var user models.User
		database.DB.First(&user, session.UserID)

		result[i] = gin.H{
			"session_id":        session.ID,
			"user_id":           session.UserID,
			"username":          user.Username,
			"task_id":           session.TaskID,
			"container_id":      session.ContainerID,
			"status":            session.Status,
			"created_at":        session.CreatedAt,
			"expires_at":        session.ExpiresAt,
			"remaining_seconds": session.ExpiresAt.Unix() - session.CreatedAt.Unix(),
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetActivityLogs returns recent activity logs (admin only)
func GetActivityLogs(c *gin.Context) {
	var logs []models.ActivityLog

	// Get last 100 activity logs
	if err := database.DB.Order("created_at DESC").Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activity logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetResources returns container resource usage (admin only)
func GetResources(c *gin.Context) {
	// TODO: Implement resource monitoring
	c.JSON(http.StatusOK, []interface{}{})
}

// TerminateSession terminates a session (admin only)
func TerminateSession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get session info before deleting
	session, _ := sessionManager.GetSession(c.Request.Context(), sessionID)

	if err := sessionManager.DeleteSession(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Log activity
	adminID := c.GetUint("user_id")
	adminUsername, _ := c.Get("username")
	details := "Admin terminated session: " + sessionID
	if session != nil {
		details = fmt.Sprintf("Admin terminated session for user %d (task: %s)", session.UserID, session.TaskID)
	}
	activityLog := models.ActivityLog{
		UserID:    adminID,
		Username:  adminUsername.(string),
		Action:    "admin_terminate_session",
		Details:   details,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	database.DB.Create(&activityLog)

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

// DeleteUserByID deletes a user and all related data (admin only)
func DeleteUserByID(c *gin.Context) {
	userID := c.Param("id")

	// Parse user ID
	var uid uint
	if _, err := fmt.Sscanf(userID, "%d", &uid); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get user info before deletion for logging
	var user models.User
	if err := database.DB.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Use transaction to delete all related data
	err := database.WithTransaction(c.Request.Context(), func(tx *gorm.DB) error {
		// Delete activity logs
		if err := tx.Where("user_id = ?", uid).Delete(&models.ActivityLog{}).Error; err != nil {
			return fmt.Errorf("failed to delete activity logs: %w", err)
		}

		// Delete progress records
		if err := tx.Where("user_id = ?", uid).Delete(&models.Progress{}).Error; err != nil {
			return fmt.Errorf("failed to delete progress: %w", err)
		}

		// Delete sessions
		if err := tx.Where("user_id = ?", uid).Delete(&models.Session{}).Error; err != nil {
			return fmt.Errorf("failed to delete sessions: %w", err)
		}

		// Finally delete the user
		if err := tx.Delete(&models.User{}, uid).Error; err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Uint("user_id", uid).Msg("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete user",
			"message": "An error occurred while deleting the user and related data",
		})
		return
	}

	// Log activity (after successful deletion)
	adminID := c.GetUint("user_id")
	adminUsername, _ := c.Get("username")
	activityLog := models.ActivityLog{
		UserID:    adminID,
		Username:  adminUsername.(string),
		Action:    "admin_delete_user",
		Details:   fmt.Sprintf("Admin deleted user: %s (ID: %d)", user.Username, uid),
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	database.DB.Create(&activityLog)

	log.Info().
		Uint("deleted_user_id", uid).
		Str("deleted_username", user.Username).
		Uint("admin_id", adminID).
		Msg("User and all related data deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "User and all related data deleted successfully",
		"deleted": gin.H{
			"id":       uid,
			"username": user.Username,
		},
	})
}
