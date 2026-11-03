package logger

import (
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LogActivity logs user activity
func LogActivity(c *gin.Context, action string, details string) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	activityLog := models.ActivityLog{
		Action:    action,
		Details:   truncate(details, 1000),
		IPAddress: c.ClientIP(),
		UserAgent: truncate(c.GetHeader("User-Agent"), 255),
	}

	if id, ok := userID.(uint); ok {
		activityLog.UserID = id
	}
	if name, ok := username.(string); ok {
		activityLog.Username = name
	}

	if database.DB == nil {
		return
	}
	if err := database.DB.Create(&activityLog).Error; err != nil {
		log.Error().Err(err).Msg("Failed to log activity")
	}
}

// LogActivityDirect logs activity without gin context
func LogActivityDirect(userID uint, username, action, details, ipAddress string) {
	activityLog := models.ActivityLog{
		UserID:    userID,
		Username:  username,
		Action:    action,
		Details:   truncate(details, 1000),
		IPAddress: ipAddress,
	}

	if database.DB == nil {
		return
	}
	if err := database.DB.Create(&activityLog).Error; err != nil {
		log.Error().Err(err).Msg("Failed to log activity")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
