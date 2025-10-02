package database

import (
	"context"
	"fmt"

	"github.com/devops-lab/backend/configs"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var RedisClient *redis.Client

// ConnectRedis establishes Redis connection
func ConnectRedis(cfg *configs.RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info().Msg("Redis connected successfully")
	return nil
}

// CloseRedis closes Redis connection
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// RedisHealthCheck checks Redis connection
func RedisHealthCheck() error {
	ctx := context.Background()
	return RedisClient.Ping(ctx).Err()
}
