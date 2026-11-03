package configs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// MinJWTSecretLength mirrors auth.MinSecretLength (kept here to avoid an import cycle).
const MinJWTSecretLength = 32

// maxJWTExpiration caps JWT_EXPIRATION so a typo cannot mint near-permanent tokens.
const maxJWTExpiration = 7 * 24 * time.Hour

// defaultTrustedProxies are loopback and RFC1918 ranges (docker bridge networks, local reverse proxy).
const defaultTrustedProxies = "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

// Bounds and default for AUTH_RATE_LIMIT_PER_MIN (login/register requests per client IP per minute).
const (
	DefaultAuthRateLimitPerMin = 10
	minAuthRateLimitPerMin     = 1
	maxAuthRateLimitPerMin     = 10000
)

// Bounds and default for SESSION_RATE_LIMIT_PER_MIN (lab creates/recreates per user per minute).
const (
	DefaultSessionRateLimitPerMin = 6
	maxSessionRateLimitPerMin     = 10000
)

// Lab resource guard defaults.
const (
	// DefaultDiskMaxBytes is LAB_DISK_MAX_BYTES: a session whose container
	// writable layer grows beyond it is stopped by the sweeper (2 GiB).
	DefaultDiskMaxBytes int64 = 2 << 30
	// DefaultIdleTimeout is LAB_IDLE_TIMEOUT: a session with no terminal
	// attached for this long is reclaimed.
	DefaultIdleTimeout = 15 * time.Minute
	minIdleTimeout     = time.Minute
)

// knownJWTPlaceholders are values shipped in examples/compose files; they must never be used in production.
var knownJWTPlaceholders = []string{
	"change-me-in-production",
	"your-super-secret-jwt-key-change-in-production",
	"changeme",
	"change-me",
	"secret",
	"jwt-secret",
	"your-secret-key",
	"supersecret",
}

// knownDBPasswords are default database passwords that must never be used in production.
var knownDBPasswords = []string{"devopslab_secret", "postgres", "password"}

// Config is the complete runtime configuration, loaded from the environment.
type Config struct {
	Server     ServerConfig
	Docker     DockerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	CORS       CORSConfig
	Monitoring MonitoringConfig
	Security   SecurityConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port        string
	GinMode     string
	Environment string
}

// DockerConfig holds Docker daemon access and lab container settings.
type DockerConfig struct {
	// Host is the Docker daemon address. DOCKER_HOST wins; otherwise "unix://" + Socket.
	Host             string
	Socket           string
	DefaultImage     string
	TasksDir         string
	SessionTTL       int
	MemLimit         int64
	CPUNano          int64
	PidsLimit        int64
	MaxSessionsTotal int

	// Runtime is the OCI runtime for lab containers (LAB_RUNTIME, e.g. "runsc"
	// for gVisor). Empty means the daemon default (runc).
	Runtime string
	// AllowNetAdmin lets tasks that request CAP_NET_ADMIN run
	// (LAB_ALLOW_NET_ADMIN). Default: true in development; in production
	// false unless Runtime is set. Such tasks are reported unavailable otherwise.
	AllowNetAdmin bool
	// StorageSize caps the container writable layer via --storage-opt size
	// (LAB_STORAGE_SIZE, e.g. "2G"). Only works on overlay2 over xfs with
	// pquota; empty disables it.
	StorageSize string
	// DiskMaxBytes is the writable-layer size above which the sweeper stops a
	// session (LAB_DISK_MAX_BYTES, default 2 GiB). Works on every storage driver.
	DiskMaxBytes int64
	// IdleTimeout reclaims sessions with no terminal attached for this long
	// (LAB_IDLE_TIMEOUT, default 15m, minimum 1m).
	IdleTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds token signing settings.
type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	// Generated is true when no secret was configured and an ephemeral one was generated (non-production only).
	Generated bool
}

// CORSConfig holds the browser origins allowed to call the API.
type CORSConfig struct {
	Origins []string
}

// MonitoringConfig holds metrics settings.
type MonitoringConfig struct {
	PrometheusEnabled bool
	PrometheusPort    string
}

// SecurityConfig holds cookie and proxy trust settings.
type SecurityConfig struct {
	// CookieSecure sets the Secure flag on the auth cookie (COOKIE_SECURE, default true).
	CookieSecure bool
	// TrustedProxies are CIDRs/IPs whose X-Forwarded-For is honoured (TRUSTED_PROXIES).
	TrustedProxies []string
	// AuthRateLimitPerMin caps login/register requests per client IP per minute
	// (AUTH_RATE_LIMIT_PER_MIN, default 10, clamped to 1..10000). Raise only for automated tests.
	AuthRateLimitPerMin int
	// SessionRateLimitPerMin caps lab creates/recreates per user per minute
	// (SESSION_RATE_LIMIT_PER_MIN, default 6, clamped to 1..10000).
	SessionRateLimitPerMin int
}

// IsProduction reports whether ENVIRONMENT=production.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Server.Environment, "production")
}

// Load reads configuration from the environment (and .env if present).
func Load() *Config {
	_ = godotenv.Load()

	env := getEnv("ENVIRONMENT", "production")

	jwtExp, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	if err != nil || jwtExp <= 0 {
		jwtExp = 24 * time.Hour
	}
	if jwtExp > maxJWTExpiration {
		jwtExp = maxJWTExpiration
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	generated := false
	if jwtSecret == "" && !strings.EqualFold(env, "production") {
		// Dev convenience only: tokens become invalid after a restart.
		jwtSecret = randomHex(32)
		generated = true
	}

	ginMode := getEnv("GIN_MODE", "release")
	if strings.EqualFold(env, "production") {
		ginMode = "release"
	}

	socket := getEnv("DOCKER_SOCKET", "/var/run/docker.sock")
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix://" + socket
	}

	labRuntime := strings.TrimSpace(os.Getenv("LAB_RUNTIME"))
	// NET_ADMIN on runc means the student's root talks to the host kernel's
	// netfilter code: allow it by default only in development or when a
	// sandboxed runtime is configured.
	allowNetAdmin := getEnvBool("LAB_ALLOW_NET_ADMIN", !strings.EqualFold(env, "production") || labRuntime != "")

	idle, err := time.ParseDuration(getEnv("LAB_IDLE_TIMEOUT", DefaultIdleTimeout.String()))
	if err != nil || idle <= 0 {
		idle = DefaultIdleTimeout
	}
	idle = max(idle, minIdleTimeout)

	diskMax := getEnvInt64("LAB_DISK_MAX_BYTES", DefaultDiskMaxBytes)
	if diskMax <= 0 {
		diskMax = DefaultDiskMaxBytes
	}

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			GinMode:     ginMode,
			Environment: env,
		},
		Docker: DockerConfig{
			Host:             dockerHost,
			Socket:           socket,
			DefaultImage:     getEnv("DEFAULT_IMAGE", "devopslab-lab:latest"),
			TasksDir:         getEnv("TASKS_DIR", "./tasks"),
			SessionTTL:       getEnvInt("SESSION_TTL", 3600),
			MemLimit:         getEnvInt64("MEM_LIMIT", 536870912),
			CPUNano:          getEnvInt64("CPU_NANO", 500000000),
			PidsLimit:        getEnvInt64("PIDS_LIMIT", 256),
			MaxSessionsTotal: getEnvInt("MAX_SESSIONS_TOTAL", 20),
			Runtime:          labRuntime,
			AllowNetAdmin:    allowNetAdmin,
			StorageSize:      strings.TrimSpace(os.Getenv("LAB_STORAGE_SIZE")),
			DiskMaxBytes:     diskMax,
			IdleTimeout:      idle,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "devopslab"),
			Password: getEnv("DB_PASSWORD", "devopslab_secret"),
			DBName:   getEnv("DB_NAME", "devopslab"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:     jwtSecret,
			Expiration: jwtExp,
			Generated:  generated,
		},
		CORS: CORSConfig{
			Origins: splitList(getEnv("CORS_ORIGINS", "http://localhost:3000")),
		},
		Monitoring: MonitoringConfig{
			PrometheusEnabled: getEnvBool("PROMETHEUS_ENABLED", true),
			PrometheusPort:    getEnv("PROMETHEUS_PORT", "9090"),
		},
		Security: SecurityConfig{
			CookieSecure:   getEnvBool("COOKIE_SECURE", true),
			TrustedProxies: splitList(getEnv("TRUSTED_PROXIES", defaultTrustedProxies)),
			AuthRateLimitPerMin: clampInt(getEnvInt("AUTH_RATE_LIMIT_PER_MIN", DefaultAuthRateLimitPerMin),
				minAuthRateLimitPerMin, maxAuthRateLimitPerMin),
			SessionRateLimitPerMin: clampInt(getEnvInt("SESSION_RATE_LIMIT_PER_MIN", DefaultSessionRateLimitPerMin),
				1, maxSessionRateLimitPerMin),
		},
	}
}

// Validate checks the configuration. In production it refuses insecure settings
// (placeholder/short JWT secret, default DB password, wildcard CORS). It returns an
// error describing every problem found; callers should treat it as fatal.
func (c *Config) Validate() error {
	var problems []string

	if c.Server.Port == "" {
		problems = append(problems, "PORT must not be empty")
	}
	switch c.Server.GinMode {
	case "release", "debug", "test":
	default:
		problems = append(problems, "GIN_MODE must be one of release, debug, test")
	}
	if c.Docker.MaxSessionsTotal < 1 {
		problems = append(problems, "MAX_SESSIONS_TOTAL must be >= 1")
	}

	// Applies everywhere: auth.InitJWT would reject it anyway.
	if len(c.JWT.Secret) < MinJWTSecretLength {
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d bytes (generate one with: openssl rand -hex 32)", MinJWTSecretLength))
	}

	if c.IsProduction() {
		if c.JWT.Secret == "" {
			problems = append(problems, "JWT_SECRET is required in production")
		} else if isPlaceholderSecret(c.JWT.Secret) {
			problems = append(problems, "JWT_SECRET is a known placeholder value; set a random secret (openssl rand -hex 32)")
		}
		for _, p := range knownDBPasswords {
			if c.Database.Password == p {
				problems = append(problems, fmt.Sprintf("DB_PASSWORD is a known default (%q); set a strong password", p))
				break
			}
		}
		if len(c.CORS.Origins) == 0 {
			problems = append(problems, "CORS_ORIGINS must list the public frontend origin(s) in production")
		}
		for _, o := range c.CORS.Origins {
			if strings.Contains(o, "*") {
				problems = append(problems, "CORS_ORIGINS must not contain '*' in production (credentials are used)")
				break
			}
		}
	}

	if len(problems) > 0 {
		return errors.New("insecure or invalid configuration (ENVIRONMENT=" + c.Server.Environment + "):\n  - " + strings.Join(problems, "\n  - "))
	}

	if c.JWT.Generated {
		log.Warn().Msg("JWT_SECRET not set: using an ephemeral random secret (non-production only); sessions reset on restart")
	}
	if c.Docker.AllowNetAdmin && c.Docker.Runtime == "" && c.IsProduction() {
		log.Warn().Msg("LAB_ALLOW_NET_ADMIN=true without LAB_RUNTIME: students get CAP_NET_ADMIN against the host kernel (use gVisor, see SECURITY.md)")
	}
	if !c.Docker.AllowNetAdmin {
		log.Info().Msg("LAB_ALLOW_NET_ADMIN=false: tasks that need CAP_NET_ADMIN are reported unavailable")
	}
	if !c.Security.CookieSecure {
		log.Warn().Msg("COOKIE_SECURE=false: auth cookie will be sent over plain HTTP (local development only)")
	}
	log.Info().Str("port", c.Server.Port).Str("env", c.Server.Environment).Msg("Server config")
	log.Info().Str("db_host", c.Database.Host).Str("db_name", c.Database.DBName).Msg("Database config")
	log.Info().Str("redis_host", c.Redis.Host).Msg("Redis config")
	return nil
}

func isPlaceholderSecret(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	for _, p := range knownJWTPlaceholders {
		if lower == p {
			return true
		}
	}
	return strings.Contains(lower, "change-me") ||
		strings.Contains(lower, "changeme") ||
		strings.Contains(lower, "change-in-production")
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func clampInt(v, lo, hi int) int {
	return min(max(v, lo), hi)
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimRight(strings.TrimSpace(p), "/"); t != "" {
			result = append(result, t)
		}
	}
	return result
}
