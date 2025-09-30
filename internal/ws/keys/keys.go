package keys

import (
	"sync"

	"go.uber.org/zap"
	
	"signaling-server/internal/logger"
)

type keyStore struct {
	sync.RWMutex
	keys map[string]map[string][]byte
}

var store = &keyStore{
	keys: make(map[string]map[string][]byte),
}

func SetAESKey(roomKey, clientID string, key []byte) {
	if roomKey == "" || clientID == "" {
		logger.Log.Warn("invalid roomKey or clientID", zap.String("roomKey", roomKey), zap.String("clientID", clientID))
		return
	}

	store.Lock()
	defer store.Unlock()
	if store.keys[roomKey] == nil {
		store.keys[roomKey] = make(map[string][]byte)
	}
	store.keys[roomKey][clientID] = key
	logger.Log.Debug("set AES key", zap.String("room", roomKey), zap.String("client", clientID))
}

func GetAESKey(roomKey, clientID string) ([]byte, bool) {
	if roomKey == "" || clientID == "" {
		logger.Log.Warn("invalid roomKey or clientID", zap.String("roomKey", roomKey), zap.String("clientID", clientID))
		return nil, false
	}

	store.RLock()
	defer store.RUnlock()
	room, ok := store.keys[roomKey]
	if !ok {
		return nil, false
	}
	key, ok := room[clientID]
	return key, ok
}

func DeleteAESKey(roomKey, clientID string) {
	if roomKey == "" || clientID == "" {
		logger.Log.Warn("invalid roomKey or clientID", zap.String("roomKey", roomKey), zap.String("clientID", clientID))
		return
	}

	store.Lock()
	defer store.Unlock()
	if room, exists := store.keys[roomKey]; exists {
		delete(room, clientID)
		if len(room) == 0 {
			delete(store.keys, roomKey)
			logger.Log.Debug("deleted empty room", zap.String("room", roomKey))
		}
		logger.Log.Debug("deleted AES key", zap.String("room", roomKey), zap.String("client", clientID))
	}
}

func GetClientsInRoom(roomKey string) []string {
	if roomKey == "" {
		logger.Log.Warn("invalid roomKey", zap.String("roomKey", roomKey))
		return nil
	}

	store.RLock()
	defer store.RUnlock()
	var clients []string
	if room, ok := store.keys[roomKey]; ok {
		for clientID := range room {
			clients = append(clients, clientID)
		}
	}
	logger.Log.Debug("retrieved clients in room", zap.String("room", roomKey), zap.Int("count", len(clients)))
	return clients
}
