package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"awesomeProject/internal/service"

	"github.com/gin-gonic/gin"
)

// SSEController SSE 专用控制器（独立于 UserHandler）
type SSEController struct {
	sseService *service.SSEService
}

// NewSSEController SSE 控制器的工厂函数（无变更）
func NewSSEController(sseService *service.SSEService) *SSEController {
	return &SSEController{
		sseService: sseService,
	}
}

// SSEHandler SSE 端点：订阅并持续接收每 2 秒推送的当前时间
func (sc *SSEController) SSEHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	ctx := c.Request.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	flusher, _ := c.Writer.(http.Flusher)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
			}
			jsonData, err := json.Marshal(payload)
			if err != nil {
				return
			}
			eventData := "data: " + string(jsonData) + "\n\n"
			if _, err := c.Writer.Write([]byte(eventData)); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
