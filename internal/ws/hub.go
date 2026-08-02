package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type        string `json:"type"`
	TaskID      string `json:"task_id,omitempty"`
	Filename    string `json:"filename,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

type userConn struct {
	conn *websocket.Conn
}

type Hub struct {
	mu    sync.RWMutex
	conns map[uint]*userConn
}

func NewHub() *Hub {
	return &Hub{conns: make(map[uint]*userConn)}
}

func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.conns[userID]; ok {
		old.conn.Close()
	}
	h.conns[userID] = &userConn{conn: conn}
}

func (h *Hub) Unregister(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, userID)
}

func (h *Hub) Send(userID uint, msg Message) error {
	h.mu.RLock()
	uc, ok := h.conns[userID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	uc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return uc.conn.WriteJSON(msg)
}
