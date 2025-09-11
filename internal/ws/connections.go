package ws

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

var conns = struct {
	sync.RWMutex
	m map[string]*websocket.Conn
}{m: make(map[string]*websocket.Conn)}

func addConnection(id string, c *websocket.Conn) {
	conns.Lock()
	defer conns.Unlock()
	conns.m[id] = c
}

func removeConnection(id string) {
	conns.Lock()
	defer conns.Unlock()
	delete(conns.m, id)
}

func getConnection(id string) *websocket.Conn {
	conns.RLock()
	defer conns.RUnlock()
	return conns.m[id]
}
