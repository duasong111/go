package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func InitRedis() {
	Client = redis.NewClient(&redis.Options{
		Addr:     "192.168.18.204:6379",
		Password: "gsm200818534",
		DB:       0,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     8,
		MinIdleConns: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 測試連線
	pong, err := Client.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis 連線失敗: %v", err)
	}

	log.Printf("Redis 連線成功！PING → %s", pong)

	// 可選：順便寫入/讀取一次測試key（驗證寫入權限）
	testKey := "test:ping:" + time.Now().Format("20060102150405")
	err = Client.Set(ctx, testKey, "ok", 60*time.Second).Err()
	if err != nil {
		log.Printf("Redis 寫入測試失敗: %v", err)
		return
	}

	val, err := Client.Get(ctx, testKey).Result()
	if err != nil || val != "ok" {
		log.Printf("Redis 讀取測試失敗: %v (got: %q)", err, val)
		return
	}

	log.Println("Redis 讀寫測試通過 ✓")
}
