package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"sync"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type pubsubStore struct {
	pubsub *redis.PubSub
	mu     sync.RWMutex
}

var pubsub = &pubsubStore{}

func SubscribeRoomEvents(cfg *config.RedisConfig, handler func(channel string, message []byte)) error {
	client, err := GetClient()
	if err != nil {
		return err
	}

	pubsub.mu.Lock()
	pubsub.pubsub = client.Subscribe(context.Background(), cfg.PubSubChannel)
	pubsub.mu.Unlock()

	if _, err := pubsub.pubsub.Receive(context.Background()); err != nil {
		logger.Log.Error("failed to subscribe to Redis channel", zap.String("channel", cfg.PubSubChannel), zap.Error(err))
		return fmt.Errorf("redis: failed to subscribe to channel %s: %w", cfg.PubSubChannel, err)
	}

	ch := pubsub.pubsub.Channel()
	go func() {
		for msg := range ch {
			handler(msg.Channel, []byte(msg.Payload))
		}
	}()

	logger.Log.Info("subscribed to Redis channel", zap.String("channel", cfg.PubSubChannel))
	return nil
}

func PublishRoomEvent(channel, eventType, roomKey string, payload interface{}) error {
	client, err := GetClient()
	if err != nil {
		return err
	}

	msg := map[string]interface{}{
		"type": eventType,
		"room": roomKey,
		"data": payload,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		logger.Log.Error("failed to marshal pubsub message", zap.Error(err))
		return fmt.Errorf("redis: failed to marshal pubsub message: %w", err)
	}

	if err := client.Publish(context.Background(), channel, b).Err(); err != nil {
		logger.Log.Error("failed to publish pubsub message", zap.String("channel", channel), zap.Error(err))
		return fmt.Errorf("redis: failed to publish to channel %s: %w", channel, err)
	}

	return nil
}

func Unsubscribe() error {
	pubsub.mu.RLock()
	if pubsub.pubsub == nil {
		pubsub.mu.RUnlock()
		return nil
	}
	pubsub.mu.RUnlock()

	pubsub.mu.Lock()
	defer pubsub.mu.Unlock()
	if err := pubsub.pubsub.Close(); err != nil {
		logger.Log.Error("failed to unsubscribe from Redis", zap.Error(err))
		return fmt.Errorf("redis: failed to unsubscribe: %w", err)
	}
	logger.Log.Info("unsubscribed from Redis channel")
	return nil
}

func InitPubSub(cfg *config.RedisConfig) error {
	return SubscribeRoomEvents(cfg, func(channel string, message []byte) {
		if err := HandlePubSubSignal(message); err != nil {
			logger.Log.Error("failed to handle pubsub signal", zap.Error(err))
		}
	})
}
