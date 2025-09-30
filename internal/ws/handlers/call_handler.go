package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/ws/room"
	"signaling-server/internal/ws/types"
)

func handleCall(cfg *config.Config, keyPhrase, clientID, name string, raw []byte, parsed types.SignalMessage) error {
	roomKey := "room:" + keyPhrase
	callEndKey := roomKey + ":call_end_ids"

	switch parsed.Type {
	case "call_initiate", "call_accept", "call_end":
		var enriched map[string]interface{}
		if err := json.Unmarshal(raw, &enriched); err != nil {
			logger.Log.Error("failed to unmarshal call message", zap.Error(err))
			return fmt.Errorf("call_handler: failed to unmarshal message: %w", err)
		}
		enriched["from"] = name
		out, err := json.Marshal(enriched)
		if err != nil {
			logger.Log.Error("failed to marshal enriched message", zap.Error(err))
			return fmt.Errorf("call_handler: failed to marshal enriched message: %w", err)
		}

		if parsed.Type == "call_end" && parsed.CallId != "" {
			client, err := redis.GetClient()
			if err != nil {
				logger.Log.Error("failed to get Redis client", zap.Error(err))
				return fmt.Errorf("call_handler: failed to get Redis client: %w", err)
			}
			ctx := context.Background()
			exists, err := client.SIsMember(ctx, callEndKey, parsed.CallId).Result()
			if err != nil {
				logger.Log.Error("failed to check call_end_ids", zap.String("callId", parsed.CallId), zap.Error(err))
				return fmt.Errorf("call_handler: failed to check call_end_ids: %w", err)
			}
			if exists {
				logger.Log.Info("duplicate call_end", zap.String("callId", parsed.CallId))
				return nil
			}
			if err := client.SAdd(ctx, callEndKey, parsed.CallId).Err(); err != nil {
				logger.Log.Error("failed to add to call_end_ids", zap.String("callId", parsed.CallId), zap.Error(err))
				return fmt.Errorf("call_handler: failed to add to call_end_ids: %w", err)
			}
			if err := client.Expire(ctx, callEndKey, cfg.Redis.TTL).Err(); err != nil {
				logger.Log.Error("failed to set TTL for call_end_ids", zap.String("callId", parsed.CallId), zap.Error(err))
				return fmt.Errorf("call_handler: failed to set TTL for call_end_ids: %w", err)
			}
		}

		room.BroadcastToRoom(keyPhrase, clientID, out)
	}

	return nil
}
