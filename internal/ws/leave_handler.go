package ws

import (
	"encoding/json"
	"fmt"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"

	"go.uber.org/zap"
)

func handleLeave(keyPhrase, clientID, name string) {
	roomKey := "room:" + keyPhrase
	roomIDKey := roomKey + ":id"

	roomID, _ := redis.GetKey(roomIDKey)

	_ = redis.GetClient().LRem(redis.Ctx(), roomKey, 0, clientID).Err()
	removeConnection(clientID)

	left := map[string]interface{}{
		"type":   "status",
		"msg":    fmt.Sprintf("%s left", name),
		"roomId": roomID,
	}
	if b, _ := json.Marshal(left); b != nil {
		broadcastToRoom(keyPhrase, clientID, b)
	}

	logger.Log.Info("Client left",
		zap.String("room", keyPhrase),
		zap.String("client", clientID))
}
