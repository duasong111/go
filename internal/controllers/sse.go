package controllers

import (
	"awesomeProject/internal/service"
	"encoding/json"
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

// SSEHandler SSE 端点处理器（重构：使用手动循环 + Flush）
func (sc *SSEController) SSEHandler(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	ctx := c.Request.Context()
	// 从服务层获取数据通道
	dataCh := sc.sseService.GenerateRandomSSEData(ctx)

	for {
		select {
		case data, ok := <-dataCh:
			if !ok {
				return // 通道关闭，结束流
			}
			// 序列化数据
			jsonData, err := json.Marshal(data)
			if err != nil {
				return // 序列化失败，结束
			}
			eventData := "data: " + string(jsonData) + "\n\n"
			_, err = c.Writer.Write([]byte(eventData))
			if err != nil {
				return
			}
			c.Writer.Flush() // 立即刷新，确保实时推送
		case <-ctx.Done():
			return // 客户端断开
		}
	}
}
