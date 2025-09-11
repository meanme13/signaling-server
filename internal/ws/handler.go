package ws

import (
	"encoding/json"
	"fmt"
	"signaling-server/internal/logger"
	"strconv"
	"time"

	"signaling-server/internal/redis"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func Handler(c *websocket.Conn) {
	defer func() { _ = c.Close() }()

	// --- Первое сообщение от клиента ---
	var init InitMessage
	if err := c.ReadJSON(&init); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","msg":"invalid initial message"}`))
		return
	}

	if err := validate.Struct(init); err != nil {
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","msg":"invalid keyPhrase or name"}`))
		return
	}

	keyPhrase := init.KeyPhrase
	name := init.Name
	if name == "" {
		name = fmt.Sprintf("user-%d", time.Now().Unix()%10000)
	}
	limit := init.Limit
	if limit == 0 {
		limit = 2
	}

	roomKey := "room:" + keyPhrase
	roomIDKey := roomKey + ":id"
	ownerKey := roomKey + ":owner"
	limitKey := roomKey + ":limit"

	// --- Room ID ---
	roomID, err := redis.GetKey(roomIDKey)
	isInitiator := false
	if err != nil || roomID == "" {
		// создаем комнату
		roomID = fmt.Sprintf("room-%04d", time.Now().Unix()%10000)
		_ = redis.SetKey(roomIDKey, roomID, time.Hour)
		_ = redis.SetKey(ownerKey, name, time.Hour)
		_ = redis.SetKey(limitKey, fmt.Sprintf("%d", limit), time.Hour)
		isInitiator = true
	}

	// --- Уникальный ID клиента ---
	clientID := uuid.New().String()
	addConnection(clientID, c)

	// --- Добавляем в Redis ---
	clientMeta := map[string]string{"id": clientID, "name": name}
	metaJSON, _ := json.Marshal(clientMeta)
	_ = redis.GetClient().RPush(redis.Ctx(), roomKey, metaJSON).Err()
	_ = redis.GetClient().Expire(redis.Ctx(), roomKey, time.Hour).Err()

	// --- Отправляем info клиенту ---
	infoMsg := map[string]interface{}{
		"type": "info",
		"msg": func() string {
			if isInitiator {
				return "room_created"
			} else {
				return "joined"
			}
		}(),
		"roomId":    roomID,
		"initiator": isInitiator,
	}
	if b, err := json.Marshal(infoMsg); err == nil {
		_ = c.WriteMessage(websocket.TextMessage, b)
		if !isInitiator {
			// Broadcast joined to others (initiator)
			broadcastToRoom(keyPhrase, clientID, b)

			// Send pending signals to new joiner
			pendingSignalsKey := roomKey + ":pending_signals"
			pending, err := redis.GetClient().LRange(redis.Ctx(), pendingSignalsKey, 0, -1).Result()
			if err == nil && len(pending) > 0 {
				for _, sig := range pending {
					if err := c.WriteMessage(websocket.TextMessage, []byte(sig)); err != nil {
						logger.Log.Error("failed to send pending signal", zap.Error(err), zap.String("client", clientID))
					}
				}
				// Clear pending after sending
				_ = redis.GetClient().Del(redis.Ctx(), pendingSignalsKey).Err()
			}
		}
	}

	// --- Цикл приёма сообщений ---
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			break
		}

		var parsed SignalMessage
		if err := json.Unmarshal(msg, &parsed); err != nil {
			continue
		}

		switch parsed.Type {
		case "msg", "offer", "answer", "candidate", "call_start", "call_end":
			broadcastToRoom(keyPhrase, clientID, msg)

		case "signal":
			sentCount := broadcastToRoom(keyPhrase, clientID, msg)
			// If no one received it (room not full), save as pending
			limitStr, _ := redis.GetKey(limitKey)
			currentLimit, _ := strconv.Atoi(limitStr)
			items, _ := redis.GetClient().LRange(redis.Ctx(), roomKey, 0, -1).Result()
			if sentCount == 0 && len(items) < currentLimit {
				pendingSignalsKey := roomKey + ":pending_signals"
				_ = redis.GetClient().RPush(redis.Ctx(), pendingSignalsKey, string(msg)).Err()
				_ = redis.GetClient().Expire(redis.Ctx(), pendingSignalsKey, time.Hour).Err()
				logger.Log.Info("Saved pending signal", zap.String("room", keyPhrase))
			}

		case "update_limit":
			owner, _ := redis.GetKey(ownerKey)
			if owner == name {
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
				_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"warning","msg":"only owner can update limit"}`))
			}
		}
	}

	// --- Disconnect ---
	removeConnection(clientID)
	_ = redis.GetClient().LRem(redis.Ctx(), roomKey, 0, metaJSON).Err()

	leftMsg := map[string]interface{}{
		"type":   "status",
		"msg":    fmt.Sprintf("%s left", name),
		"roomId": roomID,
	}
	if b, _ := json.Marshal(leftMsg); b != nil {
		broadcastToRoom(keyPhrase, clientID, b)
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}
