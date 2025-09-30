package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"signaling-server/internal/config"
	"signaling-server/internal/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type redisStore struct {
	client *redis.Client
	mu     sync.RWMutex
}

var store = &redisStore{}

func Init(cfg *config.RedisConfig) error {
	if cfg.Host == "" {
		logger.Log.Error("Redis host is empty")
		return fmt.Errorf("redis: host cannot be empty")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Log.Error("failed to connect to Redis", zap.Error(err))
		return fmt.Errorf("redis: failed to connect to Redis: %w", err)
	}

	store.mu.Lock()
	store.client = client
	store.mu.Unlock()

	logger.Log.Info("Redis client initialized", zap.String("host", cfg.Host), zap.Int("db", cfg.DB))
	return nil
}

func GetClient() (*redis.Client, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.client == nil {
		return nil, fmt.Errorf("redis: client not initialized")
	}
	return store.client, nil
}

func SetKey(ctx context.Context, key, value string, ttl time.Duration) error {
	client, err := GetClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Set(ctx, key, value, ttl).Err(); err != nil {
		logger.Log.Error("failed to set key in Redis", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("redis: failed to set key %s: %w", key, err)
	}
	return nil
}

func GetKey(ctx context.Context, key string) (string, error) {
	client, err := GetClient()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		logger.Log.Error("failed to get key from Redis", zap.String("key", key), zap.Error(err))
		return "", fmt.Errorf("redis: failed to get key %s: %w", key, err)
	}
	return val, nil
}

func DeleteKey(ctx context.Context, key string) error {
	client, err := GetClient()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Del(ctx, key).Err(); err != nil {
		logger.Log.Error("failed to delete key from Redis", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("redis: failed to delete key %s: %w", key, err)
	}
	return nil
}
