package redis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"signaling-server/internal/crypto"
	"signaling-server/internal/logger"
	"signaling-server/internal/ws/connections"
	"signaling-server/internal/ws/keys"

	"go.uber.org/zap"
)

func HandlePubSubSignal(msg []byte) error {
	var event map[string]interface{}
	if err := json.Unmarshal(msg, &event); err != nil {
		logger.Log.Error("failed to unmarshal pubsub message", zap.Error(err))
		return fmt.Errorf("redis: failed to unmarshal pubsub message: %w", err)
	}

	roomKey, ok := event["room"].(string)
	if !ok {
		logger.Log.Warn("pubsub message missing room key")
		return fmt.Errorf("redis: pubsub message missing room key")
	}

	dataMap, ok := event["data"].(map[string]interface{})
	if !ok {
		logger.Log.Warn("pubsub message missing data")
		return fmt.Errorf("redis: pubsub message missing data")
	}

	dataStr, ok := dataMap["msg"].(string)
	if !ok {
		logger.Log.Warn("pubsub message missing data.msg")
		return fmt.Errorf("redis: pubsub message missing data.msg")
	}

	encryptedMsg, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		logger.Log.Error("failed to decode base64 msg", zap.Error(err))
		return fmt.Errorf("redis: failed to decode base64 msg: %w", err)
	}

	clients := keys.GetClientsInRoom(roomKey)
	for _, clientID := range clients {
		aesKey, ok := keys.GetAESKey(roomKey, clientID)
		if !ok {
			logger.Log.Warn("no AES key for client", zap.String("client", clientID))
			continue
		}

		decryptedMsg, err := crypto.DecryptAES(encryptedMsg, aesKey)
		if err != nil {
			logger.Log.Error("failed to decrypt AES msg for client", zap.String("client", clientID), zap.Error(err))
			continue
		}

		conn := connections.GetConnection(clientID)
		if conn == nil {
			logger.Log.Warn("no connection for client", zap.String("client", clientID))
			continue
		}

		if err := conn.WriteMessage(1, decryptedMsg); err != nil {
			logger.Log.Error("failed to write message to client", zap.String("client", clientID), zap.Error(err))
			continue
		}
	}

	return nil
}
