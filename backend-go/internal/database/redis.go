package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisClient is the shared Redis client, set by ConnectRedis.
var RedisClient *redis.Client

// ConnectRedis establishes the Redis connection with bounded timeouts.
func ConnectRedis(cfg *configs.RedisConfig) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolTimeout:  3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info().Msg("Redis connected successfully")
	return nil
}

// CloseRedis closes the Redis connection.
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}

// RedisHealthCheck pings Redis with a short timeout.
func RedisHealthCheck() error {
	if RedisClient == nil {
		return errors.New("redis not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return RedisClient.Ping(ctx).Err()
}
