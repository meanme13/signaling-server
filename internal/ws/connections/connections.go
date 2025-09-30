package connections

import (
	"sync"

	"github.com/gofiber/websocket/v2"
	"go.uber.org/zap"
	
	"signaling-server/internal/logger"
)

type connectionStore struct {
	sync.RWMutex
	conns map[string]*websocket.Conn
}

var store = &connectionStore{
	conns: make(map[string]*websocket.Conn),
}

func AddConnection(clientID string, c *websocket.Conn) {
	store.Lock()
	defer store.Unlock()
	store.conns[clientID] = c
	logger.Log.Debug("added WebSocket connection", zap.String("client", clientID))
}

func RemoveConnection(clientID string) {
	store.Lock()
	defer store.Unlock()
	if _, exists := store.conns[clientID]; exists {
		delete(store.conns, clientID)
		logger.Log.Debug("removed WebSocket connection", zap.String("client", clientID))
	}
}

func GetConnection(clientID string) *websocket.Conn {
	store.RLock()
	defer store.RUnlock()
	conn, exists := store.conns[clientID]
	if !exists {
		return nil
	}

	return conn
}
