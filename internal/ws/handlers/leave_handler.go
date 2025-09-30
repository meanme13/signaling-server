package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/ws/connections"
	"signaling-server/internal/ws/room"
)

func handleLeave(keyPhrase, clientID, name string) error {
	roomKey := "room:" + keyPhrase
	roomIDKey := roomKey + ":id"
	membersKey := roomKey + ":members"

	roomID, err := redis.GetKey(context.Background(), roomIDKey)
	if err != nil {
		logger.Log.Error("failed to get roomID", zap.Error(err))
		return fmt.Errorf("leave_handler: failed to get roomID: %w", err)
	}

	client, err := redis.GetClient()
	if err != nil {
		logger.Log.Error("failed to get Redis client", zap.Error(err))
		return fmt.Errorf("leave_handler: failed to get Redis client: %w", err)
	}
	if err := client.SRem(context.Background(), membersKey, clientID).Err(); err != nil {
		logger.Log.Error("failed to remove client from members", zap.Error(err))
		return fmt.Errorf("leave_handler: failed to remove client from members: %w", err)
	}
	connections.RemoveConnection(clientID)

	remaining, err := client.SMembers(context.Background(), membersKey).Result()
	if err != nil {
		logger.Log.Error("failed to get remaining members", zap.Error(err))
		return fmt.Errorf("leave_handler: failed to get remaining members: %w", err)
	}
	if len(remaining) == 0 {
		if err := redis.DeleteKey(context.Background(), roomIDKey); err != nil {
			logger.Log.Error("failed to delete roomID", zap.Error(err))
			return fmt.Errorf("leave_handler: failed to delete roomID: %w", err)
		}
		if err := redis.DeleteKey(context.Background(), roomKey+":owner"); err != nil {
			logger.Log.Error("failed to delete owner", zap.Error(err))
			return fmt.Errorf("leave_handler: failed to delete owner: %w", err)
		}
		if err := redis.DeleteKey(context.Background(), roomKey+":limit"); err != nil {
			logger.Log.Error("failed to delete limit", zap.Error(err))
			return fmt.Errorf("leave_handler: failed to delete limit: %w", err)
		}
		if err := redis.DeleteKey(context.Background(), membersKey); err != nil {
			logger.Log.Error("failed to delete members", zap.Error(err))
			return fmt.Errorf("leave_handler: failed to delete members: %w", err)
		}
		logger.Log.Info("room deleted after last member left", zap.String("room", keyPhrase))
	}

	left := map[string]interface{}{
		"type":   "status",
		"msg":    fmt.Sprintf("%s left", name),
		"roomId": roomID,
	}
	b, err := json.Marshal(left)
	if err != nil {
		logger.Log.Error("failed to marshal leave message", zap.Error(err))
		return fmt.Errorf("leave_handler: failed to marshal leave message: %w", err)
	}
	room.BroadcastToRoom(keyPhrase, clientID, b)

	logger.Log.Info("client left",
		zap.String("room", keyPhrase),
		zap.String("client", clientID))

	return nil
}
