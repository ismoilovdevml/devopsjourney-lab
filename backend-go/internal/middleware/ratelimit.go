package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RateLimitConfig defines fixed-window rate limiting parameters.
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int // kept for compatibility; the fixed window does not use it
	KeyPrefix         string
}

// DefaultRateLimitConfig returns default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{RequestsPerMinute: 60, BurstSize: 10, KeyPrefix: "ratelimit"}
}

const rateWindow = time.Minute

// incrScript atomically increments the window counter and makes sure it has a TTL.
var incrScript = redis.NewScript(`
local c = redis.call('INCR', KEYS[1])
if c == 1 or redis.call('TTL', KEYS[1]) < 0 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return {c, redis.call('TTL', KEYS[1])}
`)

// memoryLimiter is the fallback used when Redis is unavailable. It is per-process, so the
// effective limit is multiplied by the replica count, but it never fails open.
type memoryLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*memBucket
	lastSweep time.Time
}

type memBucket struct {
	count int
	reset time.Time
}

// redisSkipUntil (unix nanos) bypasses Redis for a short period after a failure.
var redisSkipUntil atomic.Int64

var fallbackLimiter = &memoryLimiter{buckets: make(map[string]*memBucket)}

func (m *memoryLimiter) incr(key string, now time.Time) (int, time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if now.Sub(m.lastSweep) > rateWindow {
		for k, b := range m.buckets {
			if now.After(b.reset) {
				delete(m.buckets, k)
			}
		}
		m.lastSweep = now
	}
	b, ok := m.buckets[key]
	if !ok || now.After(b.reset) {
		b = &memBucket{reset: now.Add(rateWindow)}
		m.buckets[key] = b
	}
	b.count++
	return b.count, b.reset.Sub(now)
}

func (m *memoryLimiter) reset() {
	m.mu.Lock()
	m.buckets = make(map[string]*memBucket)
	m.mu.Unlock()
}

// fallback logging is throttled so a Redis outage does not flood the logs.
var (
	fallbackLogMu   sync.Mutex
	fallbackLogLast time.Time
)

func logFallback(err error) {
	fallbackLogMu.Lock()
	defer fallbackLogMu.Unlock()
	if time.Since(fallbackLogLast) < time.Minute {
		return
	}
	fallbackLogLast = time.Now()
	log.Error().Err(err).Msg("rate limiter: Redis unavailable, using in-memory fallback limiter")
}

// hit records one request for key and returns the count in the current window and time until reset.
func hit(key string) (int, time.Duration) {
	now := time.Now()
	if rc := database.RedisClient; rc != nil && now.UnixNano() >= redisSkipUntil.Load() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		res, err := incrScript.Run(ctx, rc, []string{key}, int(rateWindow.Seconds())).Int64Slice()
		if err == nil && len(res) == 2 {
			ttl := time.Duration(res[1]) * time.Second
			if ttl <= 0 {
				ttl = rateWindow
			}
			return int(res[0]), ttl
		}
		if err == nil {
			err = fmt.Errorf("unexpected script result %v", res)
		}
		// Short circuit-breaker: don't pay the Redis timeout on every request during an outage.
		redisSkipUntil.Store(now.Add(10 * time.Second).UnixNano())
		logFallback(err)
	}
	return fallbackLimiter.incr(key, now)
}

func limitHandler(limit int, keyFn func(*gin.Context) (string, bool), message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key, ok := keyFn(c)
		if !ok {
			c.Next()
			return
		}
		count, resetIn := hit(key)
		remaining := limit - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetIn).Unix(), 10))

		if count > limit {
			retry := int64(resetIn.Seconds())
			if retry < 1 {
				retry = 1
			}
			c.Header("Retry-After", strconv.FormatInt(retry, 10))
			log.Warn().Str("key", key).Str("path", c.Request.URL.Path).Msg("rate limit exceeded")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"message":     message,
				"code":        http.StatusTooManyRequests,
				"retry_after": retry,
			})
			return
		}
		c.Next()
	}
}

// AuthRateLimiter limits login/register to 10 requests per minute per client IP (shared
// bucket for both endpoints). Redis-backed, with a logged in-memory fallback.
func AuthRateLimiter() gin.HandlerFunc {
	return limitHandler(10, func(c *gin.Context) (string, bool) {
		return "ratelimit:auth:ip:" + c.ClientIP(), true
	}, "Too many authentication attempts. Please try again later.")
}

// RateLimiter limits requests per user (if authenticated) or per client IP.
func RateLimiter(cfg RateLimitConfig) gin.HandlerFunc {
	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "ratelimit"
	}
	return limitHandler(cfg.RequestsPerMinute, func(c *gin.Context) (string, bool) {
		return prefix + ":" + getIdentifier(c), true
	}, "Too many requests. Please try again later.")
}

// StrictRateLimiter creates a strict rate limiter for sensitive endpoints.
func StrictRateLimiter(requestsPerMinute int) gin.HandlerFunc {
	return RateLimiter(RateLimitConfig{RequestsPerMinute: requestsPerMinute, KeyPrefix: "strict_ratelimit"})
}

// PerUserRateLimiter limits authenticated users only; anonymous requests pass through.
func PerUserRateLimiter(requestsPerMinute int) gin.HandlerFunc {
	return limitHandler(requestsPerMinute, func(c *gin.Context) (string, bool) {
		uid, ok := c.Get("user_id")
		if !ok {
			return "", false
		}
		return fmt.Sprintf("user_ratelimit:user:%v", uid), true
	}, "You have exceeded your request quota. Please try again later.")
}

func getIdentifier(c *gin.Context) string {
	if userID, ok := c.Get("user_id"); ok {
		return fmt.Sprintf("user:%v", userID)
	}
	return "ip:" + c.ClientIP()
}
