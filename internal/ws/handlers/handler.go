package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/logger"
	"signaling-server/internal/ws/connections"
	"signaling-server/internal/ws/keys"
	"signaling-server/internal/ws/types"
)

func Handler(cfg *config.Config, c *websocket.Conn) {
	clientID := uuid.New().String()
	name := "user-" + string(rune(rand.Intn(10000)))

	var init types.InitMessage
	if err := c.ReadJSON(&init); err != nil {
		errMsg := []byte(`{"type":"error","msg":"invalid initial message"}`)
		if err := c.WriteMessage(websocket.TextMessage, errMsg); err != nil {
			logger.Log.Error("failed to send error message", zap.Error(err))
		}
		logger.Log.Error("failed to read initial JSON", zap.Error(err))
		return
	}

	if err := types.Validate.Struct(init); err != nil {
		errMsg := []byte(fmt.Sprintf(`{"type":"error","msg":"invalid init: %v"}`, err))
		if err := c.WriteMessage(websocket.TextMessage, errMsg); err != nil {
			logger.Log.Error("failed to send error message", zap.Error(err))
		}
		return
	}

	if init.Name != "" {
		name = init.Name
	}

	roomID, aesKey, _, err := handleInit(cfg, c, init, clientID)
	if err != nil {
		logger.Log.Error("failed to handle init", zap.String("client", clientID), zap.Error(err))
		return
	}

	connections.AddConnection(clientID, c)
	defer func() {
		if err := handleLeave(init.KeyPhrase, clientID, name); err != nil {
			logger.Log.Error("failed to handle leave", zap.String("client", clientID), zap.Error(err))
		}
		keys.DeleteAESKey(cfg.RoomKeyPrefix+init.KeyPhrase, clientID)
		connections.RemoveConnection(clientID)
		if err := c.Close(); err != nil {
			logger.Log.Error("failed to close WebSocket connection", zap.String("client", clientID), zap.Error(err))
		}
	}()

	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			logger.Log.Warn("failed to read WebSocket message", zap.String("client", clientID), zap.Error(err))
			break
		}

		var parsed types.SignalMessage
		if err := json.Unmarshal(msg, &parsed); err != nil {
			logger.Log.Warn("failed to unmarshal message", zap.String("client", clientID), zap.Error(err))
			continue
		}

		switch parsed.Type {
		case "msg", "offer", "answer", "candidate", "signal":
			if aesKey == nil {
				logger.Log.Warn("signal message received without AES key", zap.String("client", clientID))
				continue
			}
			if err := HandleSignal(cfg, cfg.RoomKeyPrefix+init.KeyPhrase, clientID, msg, parsed, aesKey); err != nil {
				logger.Log.Error("failed to handle signal", zap.String("client", clientID), zap.Error(err))
				continue
			}

		case "call_initiate", "call_accept", "call_end":
			if err := handleCall(cfg, init.KeyPhrase, clientID, name, msg, parsed); err != nil {
				logger.Log.Error("failed to handle call", zap.String("client", clientID), zap.Error(err))
				continue
			}

		case "update_limit":
			if err := handleUpdateLimit(cfg, init.KeyPhrase, clientID, roomID, parsed); err != nil {
				logger.Log.Error("failed to handle update_limit", zap.String("client", clientID), zap.Error(err))
				continue
			}

		default:
			logger.Log.Warn("unknown message type", zap.String("type", parsed.Type), zap.String("client", clientID))
		}
	}
}
