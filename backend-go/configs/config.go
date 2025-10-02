package configs

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Server     ServerConfig
	Docker     DockerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	CORS       CORSConfig
	Monitoring MonitoringConfig
}

type ServerConfig struct {
	Port        string
	GinMode     string
	Environment string
}

type DockerConfig struct {
	Socket       string
	DefaultImage string
	TasksDir     string
	SessionTTL   int
	MemLimit     int64
	CPUNano      int64
	PidsLimit    int64
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type CORSConfig struct {
	Origins []string
}

type MonitoringConfig struct {
	PrometheusEnabled bool
	PrometheusPort    string
}

func Load() *Config {
	// Load .env file if exists
	_ = godotenv.Load()

	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			GinMode:     getEnv("GIN_MODE", "release"),
			Environment: getEnv("ENVIRONMENT", "production"),
		},
		Docker: DockerConfig{
			Socket:       getEnv("DOCKER_SOCKET", "/var/run/docker.sock"),
			DefaultImage: getEnv("DEFAULT_IMAGE", "ubuntu:22.04"),
			TasksDir:     getEnv("TASKS_DIR", "./tasks"),
			SessionTTL:   getEnvInt("SESSION_TTL", 3600),
			MemLimit:     getEnvInt64("MEM_LIMIT", 536870912),
			CPUNano:      getEnvInt64("CPU_NANO", 500000000),
			PidsLimit:    getEnvInt64("PIDS_LIMIT", 256),
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
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			Expiration: jwtExp,
		},
		CORS: CORSConfig{
			Origins: getCORSOrigins(),
		},
		Monitoring: MonitoringConfig{
			PrometheusEnabled: getEnvBool("PROMETHEUS_ENABLED", true),
			PrometheusPort:    getEnv("PROMETHEUS_PORT", "9090"),
		},
	}
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

func getCORSOrigins() []string {
	originsStr := getEnv("CORS_ORIGINS", "http://localhost:3000")
	// Split by comma and trim spaces
	origins := strings.Split(originsStr, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (c *Config) Validate() error {
	log.Info().Msg("Configuration loaded successfully")
	log.Info().Str("port", c.Server.Port).Str("env", c.Server.Environment).Msg("Server config")
	log.Info().Str("db_host", c.Database.Host).Str("db_name", c.Database.DBName).Msg("Database config")
	log.Info().Str("redis_host", c.Redis.Host).Msg("Redis config")
	return nil
}
