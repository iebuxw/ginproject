package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message 服务端下发的 WebSocket 消息；Type 区分消息种类，其余字段按类型可选填充
type Message struct {
	Type        string `json:"type"`                  // 消息类型：heartbeat / notification / export_complete / export_failed

	// ↓ 导出结果用
	TaskID      string `json:"task_id,omitempty"`     // 导出任务 ID（导出完成/失败）
	Filename    string `json:"filename,omitempty"`    // 导出文件名
	DownloadURL string `json:"download_url,omitempty"` // 导出下载地址
	Error       string `json:"error,omitempty"`       // 导出失败原因

	// ↓ 消息通知用
	Title       string `json:"title,omitempty"`       // 消息标题（消息通知）
	Content     string `json:"content,omitempty"`     // 消息内容（消息通知）
	ID          uint   `json:"id,omitempty"`          // 消息 ID（消息通知）
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
