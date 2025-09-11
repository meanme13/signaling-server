package ws

import (
	"encoding/json"
	"fmt"
	"signaling-server/internal/logger"
	"strconv"
	"time"

	"signaling-server/internal/redis"

	"github.com/go-playground/validator/v10" // Убедитесь, что импортировано
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
		logger.Log.Error("Failed to read initial JSON", zap.Error(err))
		return
	}

	// Логируем полученные данные для отладки
	logger.Log.Info("Received init message",
		zap.String("keyPhrase", init.KeyPhrase),
		zap.String("name", init.Name),
		zap.Int("limit", init.Limit))

	if err := validate.Struct(init); err != nil {
		// Улучшенное логирование: выводим детали ошибки
		validationErrors := err.(validator.ValidationErrors)
		logger.Log.Error("Validation failed",
			zap.Errors("errors", []error{err}),
			zap.Any("details", validationErrors))

		errorMsg := fmt.Sprintf(`{"type":"error","msg":"invalid keyPhrase or name: %v"}`, err.Error())
		_ = c.WriteMessage(websocket.TextMessage, []byte(errorMsg))
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
			logger.Log.Warn("ReadMessage error", zap.Error(err), zap.String("client", clientID))
			break
		}

		var parsed SignalMessage
		if err := json.Unmarshal(msg, &parsed); err != nil {
			logger.Log.Warn("Failed to unmarshal message", zap.Error(err), zap.ByteString("msg", msg))
			continue
		}

		switch parsed.Type {
		case "msg", "offer", "answer", "candidate":
			// Обогащаем сообщение полем from
			enrichedMsg := make(map[string]interface{})
			if err := json.Unmarshal(msg, &enrichedMsg); err == nil {
				enrichedMsg["from"] = name
				enrichedMsgBytes, _ := json.Marshal(enrichedMsg)
				broadcastToRoom(keyPhrase, clientID, enrichedMsgBytes)
			} else {
				broadcastToRoom(keyPhrase, clientID, msg)
			}

		case "call_initiate", "call_accept", "call_end":
			// Обогащаем сообщение полем from
			enrichedMsg := make(map[string]interface{})
			if err := json.Unmarshal(msg, &enrichedMsg); err == nil {
				enrichedMsg["from"] = name
				enrichedMsgBytes, _ := json.Marshal(enrichedMsg)
				// Broadcast только если не control-сообщение от отправителя (прерываем цикл)
				if parsed.Type != "call_end" {
					broadcastToRoom(keyPhrase, clientID, enrichedMsgBytes)
				} else {
					// Для call_end: broadcast всем, кроме отправителя
					sentCount := 0
					items, err := redis.GetClient().LRange(redis.Ctx(), roomKey, 0, -1).Result()
					if err != nil {
						logger.Log.Error("failed to read room members from redis", zap.Error(err), zap.String("room", keyPhrase))
						break
					}
					for _, raw := range items {
						var meta map[string]string
						if err := json.Unmarshal([]byte(raw), &meta); err != nil {
							continue
						}
						if meta["id"] == clientID { // Пропускаем отправителя
							continue
						}
						conn := getConnection(meta["id"])
						if conn == nil {
							continue
						}
						if err := conn.WriteMessage(websocket.TextMessage, enrichedMsgBytes); err != nil {
							logger.Log.Error("broadcast write error", zap.Error(err), zap.String("client", meta["id"]), zap.String("room", keyPhrase))
						} else {
							sentCount++
						}
					}
					logger.Log.Info("Broadcast call_end (excluding sender)", zap.String("room", keyPhrase), zap.Int("sent", sentCount))
				}
				// НЕ сохраняем call_end в pending (избегаем накопления)
			} else {
				broadcastToRoom(keyPhrase, clientID, msg)
			}

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
