package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/ws/connections"
	"signaling-server/internal/ws/room"
	"signaling-server/internal/ws/types"
)

func handleUpdateLimit(cfg *config.Config, keyPhrase, clientID, roomID string, parsed types.SignalMessage) error {
	roomKey := "room:" + keyPhrase
	ownerKey := roomKey + ":owner"
	limitKey := roomKey + ":limit"

	owner, err := redis.GetKey(context.Background(), ownerKey)
	if err != nil {
		logger.Log.Error("failed to get owner", zap.Error(err))
		return fmt.Errorf("limit_handler: failed to get owner: %w", err)
	}

	if owner == clientID {
		if err := redis.SetKey(context.Background(), limitKey, fmt.Sprintf("%d", parsed.Limit), cfg.Redis.TTL); err != nil {
			logger.Log.Error("failed to set room limit", zap.Error(err))
			return fmt.Errorf("limit_handler: failed to set room limit: %w", err)
		}

		info := map[string]interface{}{
			"type":   "status",
			"msg":    fmt.Sprintf("room limit updated to %d", parsed.Limit),
			"roomId": roomID,
		}
		b, err := json.Marshal(info)
		if err != nil {
			logger.Log.Error("failed to marshal limit update message", zap.Error(err))
			return fmt.Errorf("limit_handler: failed to marshal limit update message: %w", err)
		}
		room.BroadcastToRoom(keyPhrase, "", b)
	} else {
		warning := []byte(`{"type":"warning","msg":"only owner can update limit"}`)
		conn := connections.GetConnection(clientID)
		if conn != nil {
			if err := conn.WriteMessage(websocket.TextMessage, warning); err != nil {
				logger.Log.Error("failed to send warning message", zap.Error(err))
				return fmt.Errorf("limit_handler: failed to send warning message: %w", err)
			}
		}
	}

	return nil
}
