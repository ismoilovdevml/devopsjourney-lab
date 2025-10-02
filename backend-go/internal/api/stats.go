package api

import (
	"net/http"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
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
	var stats models.StatsResponse

	// Total users
	database.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	// Active sessions
	database.DB.Model(&models.Session{}).Where("status = ?", "active").Count(&stats.ActiveSessions)

	// Total attempts
	database.DB.Model(&models.Progress{}).Count(&stats.TotalAttempts)

	// Completion rate
	var passed int64
	database.DB.Model(&models.Progress{}).Where("passed = ?", true).Count(&passed)
	if stats.TotalAttempts > 0 {
		stats.CompletionRate = float64(passed) / float64(stats.TotalAttempts) * 100
	}

	c.JSON(http.StatusOK, stats)
}

// GetUsers returns all users (admin only)
func GetUsers(c *gin.Context) {
	var users []models.User

	if err := database.DB.Select("id, username, email, role, active, created_at").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUser updates user details (admin only)
func UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		Role   string `json:"role"`
		Active bool   `json:"active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", userID).Updates(req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

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
