package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// LoggerMiddleware logs HTTP requests
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		logEvent := log.Info()
		if statusCode >= 400 {
			logEvent = log.Warn()
		}
		if statusCode >= 500 {
			logEvent = log.Error()
		}

		logEvent.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", statusCode).
			Dur("duration", duration).
			Str("ip", c.ClientIP()).
			Msg("HTTP request")
	}
}
