package handlers

import (
	"encoding/base64"
	"fmt"

	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/crypto"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/ws/room"
	"signaling-server/internal/ws/types"
)

func HandleSignal(cfg *config.Config, roomKey, senderID string, msg []byte, parsed types.SignalMessage, aesKey []byte) error {
	encryptedMsg, err := crypto.EncryptAES(msg, aesKey)
	if err != nil {
		logger.Log.Error("failed to encrypt message", zap.Error(err))
		return fmt.Errorf("signal_handler: failed to encrypt message: %w", err)
	}

	sent := room.BroadcastToRoom(roomKey, senderID, encryptedMsg)
	logger.Log.Debug("signal broadcast locally", zap.String("room", roomKey), zap.Int("sent", sent))

	pubPayload := map[string]interface{}{
		"sender": senderID,
		"type":   parsed.Type,
		"msg":    base64.StdEncoding.EncodeToString(encryptedMsg),
	}
	if err := redis.PublishRoomEvent(cfg.Redis.PubSubChannel, "signal", roomKey, pubPayload); err != nil {
		logger.Log.Error("failed to publish room event", zap.Error(err))
		return fmt.Errorf("signal_handler: failed to publish room event: %w", err)
	}

	return nil
}
