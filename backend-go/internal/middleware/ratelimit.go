package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RateLimitConfig defines rate limiting parameters
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	KeyPrefix         string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		KeyPrefix:         "ratelimit",
	}
}

// RateLimiter creates a rate limiting middleware
func RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get identifier (user ID if authenticated, otherwise IP)
		identifier := getIdentifier(c)
		key := fmt.Sprintf("%s:%s", cfg.KeyPrefix, identifier)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Check and increment rate limit
		allowed, remaining, resetTime, err := checkRateLimit(ctx, key, cfg)
		if err != nil {
			log.Error().
				Err(err).
				Str("identifier", identifier).
				Msg("Rate limit check failed")
			// Allow request on error to prevent blocking on Redis issues
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

		if !allowed {
			retryAfter := resetTime - time.Now().Unix()
			if retryAfter < 0 {
				retryAfter = 60
			}

			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))

			log.Warn().
				Str("identifier", identifier).
				Str("ip", c.ClientIP()).
				Str("path", c.Request.URL.Path).
				Msg("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     fmt.Sprintf("Too many requests. Please try again in %d seconds.", retryAfter),
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// StrictRateLimiter creates a strict rate limiter for sensitive endpoints
func StrictRateLimiter(requestsPerMinute int) gin.HandlerFunc {
	return RateLimiter(RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		BurstSize:         requestsPerMinute / 6,
		KeyPrefix:         "strict_ratelimit",
	})
}

// getIdentifier returns the rate limit identifier for a request
func getIdentifier(c *gin.Context) string {
	// Try to get user ID from context (if authenticated)
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}

	// Fall back to IP address
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// checkRateLimit checks if a request is allowed and increments the counter
func checkRateLimit(ctx context.Context, key string, cfg RateLimitConfig) (allowed bool, remaining int, resetTime int64, err error) {
	now := time.Now()
	resetTime = now.Add(time.Minute).Unix()

	// Get current count
	count, err := database.RedisClient.Get(ctx, key).Int()
	if err != nil && err.Error() != "redis: nil" {
		return false, 0, resetTime, err
	}

	// Check if limit exceeded
	if count >= cfg.RequestsPerMinute {
		// Get TTL to determine reset time
		ttl, _ := database.RedisClient.TTL(ctx, key).Result()
		if ttl > 0 {
			resetTime = now.Add(ttl).Unix()
		}
		return false, 0, resetTime, nil
	}

	// Increment counter
	pipe := database.RedisClient.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Minute)
	_, err = pipe.Exec(ctx)

	if err != nil {
		return false, 0, resetTime, err
	}

	newCount := int(incrCmd.Val())
	remaining = cfg.RequestsPerMinute - newCount
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, resetTime, nil
}

// PerUserRateLimiter creates a rate limiter that only applies to authenticated users
func PerUserRateLimiter(requestsPerMinute int) gin.HandlerFunc {
	cfg := RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		BurstSize:         requestsPerMinute / 6,
		KeyPrefix:         "user_ratelimit",
	}

	return func(c *gin.Context) {
		// Only apply to authenticated requests
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		key := fmt.Sprintf("%s:user:%v", cfg.KeyPrefix, userID)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		allowed, remaining, resetTime, err := checkRateLimit(ctx, key, cfg)
		if err != nil {
			log.Error().
				Err(err).
				Interface("user_id", userID).
				Msg("Per-user rate limit check failed")
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime, 10))

		if !allowed {
			retryAfter := resetTime - time.Now().Unix()
			if retryAfter < 0 {
				retryAfter = 60
			}

			c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))

			log.Warn().
				Interface("user_id", userID).
				Str("path", c.Request.URL.Path).
				Msg("Per-user rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "You have exceeded your request quota. Please try again later.",
				"retry_after": retryAfter,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
