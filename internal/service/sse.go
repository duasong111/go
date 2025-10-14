package service

import (
	"context"
	"log" // 新增：用于调试日志
	"math/rand/v2"
	"time"
)

// SSEService SSE 专用服务（独立于 UserService）
type SSEService struct {
	userService *UserService // 调用了同级的东西，是操控数据库字段的内容
}

// NewSSEService SSE 服务的工厂函数（无变更）
func NewSSEService(us *UserService) *SSEService {
	return &SSEService{
		userService: us,
	}
}

type SSEData struct {
	RandomNumber int       `json:"random_number"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}

// GenerateRandomSSEData 生成随机数 SSE 数据流
func (ss *SSEService) GenerateRandomSSEData(ctx context.Context) <-chan SSEData {
	ch := make(chan SSEData)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		rng := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))

		for {
			select {
			case <-ctx.Done():
				log.Println("SSE goroutine stopped due to context cancel")
				return
			case <-ticker.C:
				data := SSEData{
					RandomNumber: rng.IntN(100) + 1,
					Message:      "随机数推送",
					Timestamp:    time.Now(),
				}
				// 调试日志：验证每次生成新数据
				log.Printf("Generated new data: random=%d, timestamp=%v", data.RandomNumber, data.Timestamp)
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
