package ws

import (
	"encoding/json"
	"fmt"
	"signaling-server/internal/redis"
	"time"
)

func handleUpdateLimit(keyPhrase, name, clientID, roomID string, parsed SignalMessage, isInitiator bool) {
	roomKey := "room:" + keyPhrase
	ownerKey := roomKey + ":owner"
	limitKey := roomKey + ":limit"

	owner, _ := redis.GetKey(ownerKey)

	if owner == clientID || isInitiator {
		_ = redis.SetKey(limitKey, fmt.Sprintf("%d", parsed.Limit), time.Hour)

		info := map[string]interface{}{
			"type":   "status",
			"msg":    fmt.Sprintf("room limit updated to %d", parsed.Limit),
			"roomId": roomID,
		}
		if b, _ := json.Marshal(info); b != nil {
			broadcastToRoom(keyPhrase, "", b)
		}
	} else {
		warning := []byte(`{"type":"warning","msg":"only owner can update limit"}`)
		if conn := getConnection(clientID); conn != nil {
			_ = conn.WriteMessage(1, warning)
		}
	}
}
