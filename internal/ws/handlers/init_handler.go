package handlers

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"

	"signaling-server/internal/config"
	"signaling-server/internal/crypto"
	"signaling-server/internal/logger"
	"signaling-server/internal/redis"
	"signaling-server/internal/utils"
	"signaling-server/internal/ws/connections"
	"signaling-server/internal/ws/keys"
	"signaling-server/internal/ws/types"
)

func handleInit(cfg *config.Config, c *websocket.Conn, init types.InitMessage, clientID string) (string, []byte, bool, error) {
	roomKey := "room:" + init.KeyPhrase
	roomIDKey := roomKey + ":id"
	ownerKey := roomKey + ":owner"
	limitKey := roomKey + ":limit"
	membersKey := roomKey + ":members"

	roomID, err := redis.GetKey(context.Background(), roomIDKey)
	isInitiator := false

	if err != nil || roomID == "" {
		roomID = fmt.Sprintf("room-%d", time.Now().Unix()%10000)
		if err := redis.SetKey(context.Background(), roomIDKey, roomID, cfg.Redis.TTL); err != nil {
			logger.Log.Error("failed to set roomID", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to set roomID: %w", err)
		}
		if err := redis.SetKey(context.Background(), ownerKey, clientID, cfg.Redis.TTL); err != nil {
			logger.Log.Error("failed to set owner", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to set owner: %w", err)
		}

		limit := init.Limit
		if limit == 0 {
			limit = cfg.DefaultRoomLimit
		}
		if err := redis.SetKey(context.Background(), limitKey, fmt.Sprintf("%d", limit), cfg.Redis.TTL); err != nil {
			logger.Log.Error("failed to set room limit", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to set room limit: %w", err)
		}
		isInitiator = true
	}

	client, err := redis.GetClient()
	if err != nil {
		logger.Log.Error("failed to get Redis client", zap.Error(err))
		return "", nil, false, fmt.Errorf("init_handler: failed to get Redis client: %w", err)
	}
	if err := client.SAdd(context.Background(), membersKey, clientID).Err(); err != nil {
		logger.Log.Error("failed to add client to members", zap.Error(err))
		return "", nil, false, fmt.Errorf("init_handler: failed to add client to members: %w", err)
	}
	if err := client.Expire(context.Background(), membersKey, cfg.Redis.TTL).Err(); err != nil {
		logger.Log.Error("failed to set TTL for members", zap.Error(err))
		return "", nil, false, fmt.Errorf("init_handler: failed to set TTL for members: %w", err)
	}

	connections.AddConnection(clientID, c)

	var aesKey []byte
	var encryptedForClientB64 string

	if init.AESKey != "" {
		encKeyBytes, err := base64.StdEncoding.DecodeString(init.AESKey)
		if err != nil {
			logger.Log.Error("failed to decode AES key", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to decode AES key: %w", err)
		}
		aesKey, err = crypto.DecryptRSA(encKeyBytes)
		if err != nil {
			logger.Log.Warn("DecryptRSA failed, generating new AES key", zap.Error(err))
			aesKey, err = crypto.GenerateAESKey()
			if err != nil {
				logger.Log.Error("failed to generate AES key", zap.Error(err))
				return "", nil, false, fmt.Errorf("init_handler: failed to generate AES key: %w", err)
			}
		}
	} else if init.ClientPubKey != "" {
		aesKey, err = crypto.GenerateAESKey()
		if err != nil {
			logger.Log.Error("failed to generate AES key", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to generate AES key: %w", err)
		}
		block, _ := pem.Decode([]byte(init.ClientPubKey))
		if block == nil {
			logger.Log.Error("failed to decode client public key")
			return "", nil, false, fmt.Errorf("init_handler: failed to decode client public key")
		}
		pubIf, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			logger.Log.Error("failed to parse client public key", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to parse client public key: %w", err)
		}
		clientPub, ok := pubIf.(*rsa.PublicKey)
		if !ok {
			logger.Log.Error("invalid client public key type")
			return "", nil, false, fmt.Errorf("init_handler: invalid client public key type")
		}
		enc, err := crypto.EncryptRSA(aesKey, clientPub)
		if err != nil {
			logger.Log.Error("failed to encrypt AES key", zap.Error(err))
			return "", nil, false, fmt.Errorf("init_handler: failed to encrypt AES key: %w", err)
		}
		encryptedForClientB64 = base64.StdEncoding.EncodeToString(enc)
	} else {
		errMsg := []byte(`{"type":"error","msg":"aesKey or clientPubKey required"}`)
		if err := c.WriteMessage(websocket.TextMessage, errMsg); err != nil {
			logger.Log.Error("failed to send error message", zap.Error(err))
		}
		return "", nil, false, fmt.Errorf("init_handler: aesKey or clientPubKey required")
	}

	keys.SetAESKey(roomKey, clientID, aesKey)

	infoMsg := map[string]interface{}{
		"type":      "info",
		"msg":       utils.IfThen(isInitiator, "room_created", "joined"),
		"roomId":    roomID,
		"initiator": isInitiator,
	}
	if encryptedForClientB64 != "" {
		infoMsg["aesKey"] = encryptedForClientB64
	}

	b, err := json.Marshal(infoMsg)
	if err != nil {
		logger.Log.Error("failed to marshal info message", zap.Error(err))
		return "", nil, false, fmt.Errorf("init_handler: failed to marshal info message: %w", err)
	}
	if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
		logger.Log.Error("failed to send info message", zap.Error(err))
		return "", nil, false, fmt.Errorf("init_handler: failed to send info message: %w", err)
	}

	logger.Log.Info("client joined room",
		zap.String("room", init.KeyPhrase),
		zap.String("client", clientID),
		zap.Bool("initiator", isInitiator))

	return roomID, aesKey, isInitiator, nil
}
