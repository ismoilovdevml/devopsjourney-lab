package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestIDHeader carries the request correlation id.
const RequestIDHeader = "X-Request-ID"

var requestIDRe = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID accepts a well-formed incoming X-Request-ID (from a proxy) or generates one,
// stores it as "request_id" in the context and echoes it in the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if !requestIDRe.MatchString(id) {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(RequestIDHeader, id)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// LoggerMiddleware logs one line per request. It deliberately never logs headers
// (Authorization, Cookie), query strings (may carry ids/tokens) or bodies.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		ev := log.Info()
		switch {
		case status >= 500:
			ev = log.Error()
		case status >= 400:
			ev = log.Warn()
		}

		if rid, ok := c.Get("request_id"); ok {
			ev = ev.Interface("request_id", rid)
		}
		if uid, ok := c.Get("user_id"); ok {
			ev = ev.Interface("user_id", uid)
		}
		if route := c.FullPath(); route != "" {
			ev = ev.Str("route", route)
		}
		if len(c.Errors) > 0 {
			ev = ev.Str("errors", c.Errors.ByType(gin.ErrorTypePrivate).String())
		}

		ev.Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Int("bytes", c.Writer.Size()).
			Dur("duration", time.Since(start)).
			Str("ip", c.ClientIP()).
			Msg("HTTP request")
	}
}
