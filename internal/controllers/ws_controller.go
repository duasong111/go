// internal/controllers/ws_controller.go
package controllers

import (
	"awesomeProject/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应校验 Origin，这里开发期先允许所有
		return true
	},
}

// WsHandler Gin 的 WebSocket 入口
func WsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	client := service.NewWsClient(conn)
	// 注册到 hub
	service.Hub.Register <- client

	welcome := []byte(`{"type":"welcome","message":"Connected to server! Waiting for data..."}`)
	client.Send <- welcome

	// 启动读写 goroutine
	go client.ReadPump()
	go client.WritePump()
}
