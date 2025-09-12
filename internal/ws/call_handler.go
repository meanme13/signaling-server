package ws

import (
	"encoding/json"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"time"

	"go.uber.org/zap"
)

func handleCall(keyPhrase, clientID, name string, raw []byte, parsed SignalMessage) {
	roomKey := "room:" + keyPhrase
	callEndKey := roomKey + ":call_end_ids"

	switch parsed.Type {
	case "call_initiate", "call_accept", "call_end":
		enriched := map[string]interface{}{}
		if err := json.Unmarshal(raw, &enriched); err == nil {
			enriched["from"] = name
			out, _ := json.Marshal(enriched)

			if parsed.Type == "call_end" && parsed.CallId != "" {
				if exists, _ := redis.GetClient().SIsMember(redis.Ctx(), callEndKey, parsed.CallId).Result(); exists {
					logger.Log.Info("Duplicate call_end", zap.String("callId", parsed.CallId))
					return
				}
				_ = redis.GetClient().SAdd(redis.Ctx(), callEndKey, parsed.CallId).Err()
				_ = redis.GetClient().Expire(redis.Ctx(), callEndKey, time.Hour).Err()
			}

			broadcastToRoom(keyPhrase, clientID, out)
		}
	}
}
