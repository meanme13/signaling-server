package ws

import (
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"strconv"

	"go.uber.org/zap"
)

func handleSignal(keyPhrase, clientID string, raw []byte, parsed SignalMessage) {
	roomKey := "room:" + keyPhrase
	limitKey := roomKey + ":limit"

	sentCount := broadcastToRoom(keyPhrase, clientID, raw)

	limitStr, _ := redis.GetKey(limitKey)
	currentLimit, _ := strconv.Atoi(limitStr)
	items, _ := redis.GetClient().LRange(redis.Ctx(), roomKey, 0, -1).Result()

	if parsed.Type == "signal" && sentCount == 0 && len(items) < currentLimit {
		pendingKey := roomKey + ":pending_signals"
		_ = redis.GetClient().RPush(redis.Ctx(), pendingKey, string(raw)).Err()
		_ = redis.GetClient().Expire(redis.Ctx(), pendingKey, 3600)
		logger.Log.Info("Saved pending signal", zap.String("room", keyPhrase))
	}
}
