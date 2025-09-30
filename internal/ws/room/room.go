package room

import (
	"context"
	"encoding/json"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"

	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/ws/connections"
)

func BroadcastToRoom(keyPhrase, senderID string, msg []byte) int {
	cut := len(msg)
	if cut > 50 {
		cut = 50
	}

	client, err := redis.GetClient()
	if err != nil {
		logger.Log.Error("failed to get Redis client", zap.Error(err))
		return 0
	}

	items, err := client.LRange(context.Background(), "room:"+keyPhrase, 0, -1).Result()
	if err != nil {
		logger.Log.Error("failed to read room members from Redis", zap.Error(err), zap.String("room", keyPhrase))
		return 0
	}

	logger.Log.Info("broadcasting to room",
		zap.String("room", keyPhrase),
		zap.String("msg_type", string(msg[:cut])),
	)

	sent := 0
	for _, raw := range items {
		var meta map[string]string
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			logger.Log.Warn("failed to unmarshal member meta", zap.Error(err))
			continue
		}
		if meta["id"] == senderID {
			continue
		}

		conn := connections.GetConnection(meta["id"])
		if conn == nil {
			logger.Log.Warn("no connection for client", zap.String("client", meta["id"]))
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			logger.Log.Error("broadcast write error", zap.Error(err), zap.String("client", meta["id"]), zap.String("room", keyPhrase))
		} else {
			sent++
		}
	}
	return sent
}
