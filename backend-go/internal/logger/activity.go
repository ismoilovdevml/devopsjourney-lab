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
		Details:   details,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	if userID != nil {
		activityLog.UserID = userID.(uint)
	}

	if username != nil {
		activityLog.Username = username.(string)
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
		Details:   details,
		IPAddress: ipAddress,
	}

	if err := database.DB.Create(&activityLog).Error; err != nil {
		log.Error().Err(err).Msg("Failed to log activity")
	}
}
