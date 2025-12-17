package service

import (
	"context"
	"log"
	"time"
)

// SSEService SSE 专用服务（独立于 UserService）
type SSEService struct {
	userService *UserService
}

// NewSSEService SSE 服务的工厂函数，初始化服务
func NewSSEService(us *UserService) *SSEService {
	return &SSEService{
		userService: us,
	}
}

type SSEData struct {
	Timestamp time.Time `json:"timestamp"`
}

// GenerateTimeSSEData 生成每 2 秒当前时间 SSE 数据流（可选扩展方法，未在控制器中使用）
func (ss *SSEService) GenerateTimeSSEData(ctx context.Context) <-chan SSEData {
	ch := make(chan SSEData)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("SSE goroutine stopped due to context cancel")
				return
			case <-ticker.C:
				data := SSEData{
					Timestamp: time.Now(),
				}
				log.Printf("Generated new data: timestamp=%v", data.Timestamp)
				select {
				case ch <- data:
				case <-ctx.Done():
					log.Println("SSE channel send canceled")
					return
				}
			}
		}
	}()
	return ch
}
