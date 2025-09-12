package ws

import (
	"encoding/json"
	"fmt"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"time"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
)

func handleInit(c *websocket.Conn, init InitMessage, clientID, name string) (string, bool) {
	roomKey := "room:" + init.KeyPhrase
	roomIDKey := roomKey + ":id"
	ownerKey := roomKey + ":owner"
	limitKey := roomKey + ":limit"
	membersKey := roomKey + ":members"

	roomID, err := redis.GetKey(roomIDKey)
	isInitiator := false

	if err != nil || roomID == "" {
		roomID = fmt.Sprintf("room-%d", time.Now().Unix()%10000)
		_ = redis.SetKey(roomIDKey, roomID, time.Hour)
		_ = redis.SetKey(ownerKey, clientID, time.Hour)

		limit := init.Limit
		if limit == 0 {
			limit = 2
		}
		_ = redis.SetKey(limitKey, fmt.Sprintf("%d", limit), time.Hour)

		isInitiator = true
	}

	_ = redis.GetClient().SAdd(redis.Ctx(), membersKey, clientID).Err()
	_ = redis.GetClient().Expire(redis.Ctx(), membersKey, time.Hour).Err()

	addConnection(clientID, c)

	infoMsg := map[string]interface{}{
		"type":      "info",
		"msg":       ifThen(isInitiator, "room_created", "joined"),
		"roomId":    roomID,
		"initiator": isInitiator,
	}
	if b, err := json.Marshal(infoMsg); err == nil {
		_ = c.WriteMessage(websocket.TextMessage, b)
	}

	logger.Log.Info("Client joined room",
		zap.String("room", init.KeyPhrase),
		zap.String("client", clientID),
		zap.Bool("initiator", isInitiator))

	return roomID, isInitiator
}

func ifThen(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
