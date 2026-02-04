// internal/service/ws_service.go
package service

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gorilla/websocket"
)

const (
	// 写超时（防止客户端卡住）
	writeWait = 10 * time.Second

	// 读超时（心跳检测用）
	pongWait = 60 * time.Second

	// 心跳发送间隔（比 pongWait 小一点）
	pingPeriod = (pongWait * 9) / 10
)

// WsHub 管理所有 WebSocket 连接
type WsHub struct {
	clients    map[*WsClient]bool // 活跃客户端集合
	broadcast  chan []byte        // 广播通道
	Register   chan *WsClient     // 注册新连接
	unregister chan *WsClient     // 注销连接
	mutex      sync.RWMutex       // 保护 clients map
}

var Hub = &WsHub{
	clients:    make(map[*WsClient]bool),
	broadcast:  make(chan []byte, 256), // 有缓冲，防止阻塞
	Register:   make(chan *WsClient),
	unregister: make(chan *WsClient),
}

// WsClient 表示一个 WebSocket 连接
type WsClient struct {
	hub  *WsHub
	conn *websocket.Conn
	Send chan []byte // 发送给客户端的消息队列
}

// NewWsClient 创建新客户端
func NewWsClient(conn *websocket.Conn) *WsClient {
	return &WsClient{
		hub:  Hub,
		conn: conn,
		Send: make(chan []byte, 256),
	}
}

// 启动 Hub 的主循环（在 main 中启动一次）
func (h *WsHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			logger.Infof("WebSocket client connected, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				logger.Infof("WebSocket client disconnected, total: %d", len(h.clients))
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// 客户端发送队列满，关闭连接
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToWS 从 mqtt.go 调用这里广播
func BroadcastToWS(topic string, payload string) {
	// 可以包装成更结构化的消息
	msg := map[string]interface{}{
		"topic":   topic,
		"payload": payload,
	}
	data, err := json.Marshal(msg)

	//Hub.mutex.RLock()
	//Hub.mutex.RUnlock()

	if err != nil {
		logger.Errorf("Failed to marshal WS broadcast message: %v", err)
		return
	}

	Hub.broadcast <- data
	//logger.Debugf("Broadcasted to WS: %s", string(data))
}

// 下面是客户端的读写 goroutine

// ReadPump 读取客户端消息（目前只处理 pong，防止超时）
func (c *WsClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512) // 防止恶意大消息
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warnf("WS read error: %v", err)
			}
			break
		}
		// 如果未来需要前端发消息给后端（如控制命令），在这里处理
	}
}

// writePump 向客户端发送消息 + 心跳
func (c *WsClient) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 一次性写完所有待发消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
