package ws

import (
	"encoding/json"
	"signaling-server/internal/logger"

	"signaling-server/internal/redis"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

func broadcastToRoom(keyPhrase, senderID string, msg []byte) int {
	items, err := redis.GetClient().LRange(redis.Ctx(), "room:"+keyPhrase, 0, -1).Result()
	if err != nil {
		logger.Log.Error("failed to read room members from redis", zap.Error(err), zap.String("room", keyPhrase))
		return 0
	}

	logger.Log.Info("Broadcasting to room", zap.String("room", keyPhrase), zap.String("msg_type", string(msg[:minInt(50, len(msg))])))

	sent := 0
	for _, raw := range items {
		var meta map[string]string
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			continue
		}
		if meta["id"] == senderID {
			continue
		}

		conn := getConnection(meta["id"])
		if conn == nil {
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

// Вспомогательная функция для min
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
