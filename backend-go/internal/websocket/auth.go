package websocket

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AuthenticateWebSocket validates JWT token and verifies session ownership
func AuthenticateWebSocket(c *gin.Context) (*models.Session, *auth.Claims, error) {
	// Get token from query parameter or Authorization header
	token := c.Query("token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		log.Warn().
			Str("ip", c.ClientIP()).
			Msg("WebSocket connection attempt without token")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "Token must be provided via 'token' query parameter or Authorization header",
		})
		return nil, nil, errors.New("no token provided")
	}

	// Validate JWT token
	claims, err := auth.ValidateToken(token)
	if err != nil {
		log.Warn().
			Err(err).
			Str("ip", c.ClientIP()).
			Msg("Invalid token for WebSocket connection")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
			"message": "The provided token is invalid or expired",
		})
		return nil, nil, fmt.Errorf("invalid token: %w", err)
	}

	// Get session ID from query parameter
	sessionID := c.Query("sessionId")
	if sessionID == "" {
		log.Warn().
			Uint("user_id", claims.UserID).
			Msg("WebSocket connection attempt without session ID")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID required",
			"message": "sessionId query parameter is required",
		})
		return nil, nil, errors.New("no session ID provided")
	}

	// Verify session exists and belongs to the user
	var session models.Session
	err = database.DB.Where("id = ? AND status = ?", sessionID, "active").
		First(&session).Error

	if err != nil {
		log.Warn().
			Err(err).
			Str("session_id", sessionID).
			Uint("user_id", claims.UserID).
			Msg("Session not found for WebSocket connection")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
			"message": "The specified session does not exist or has been stopped",
		})
		return nil, nil, fmt.Errorf("session not found: %w", err)
	}

	// Verify session ownership
	if session.UserID != claims.UserID {
		log.Warn().
			Str("session_id", sessionID).
			Uint("session_user_id", session.UserID).
			Uint("token_user_id", claims.UserID).
			Msg("Unauthorized WebSocket connection attempt")
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
			"message": "You do not have permission to access this session",
		})
		return nil, nil, errors.New("session ownership verification failed")
	}

	log.Info().
		Str("session_id", sessionID).
		Uint("user_id", claims.UserID).
		Str("username", claims.Username).
		Msg("WebSocket authentication successful")

	return &session, claims, nil
}

// CheckOriginFromConfig checks if the origin is allowed based on configuration
func CheckOriginFromConfig(allowedOrigins []string) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")

		// Allow same-origin requests
		if origin == "" {
			return true
		}

		// Check if origin is in allowed list
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				log.Debug().
					Str("origin", origin).
					Msg("WebSocket origin allowed")
				return true
			}
		}

		log.Warn().
			Str("origin", origin).
			Strs("allowed_origins", allowedOrigins).
			Msg("WebSocket origin rejected")
		return false
	}
}
