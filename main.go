package main

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/redis"
	"awesomeProject/internal/routes"
	"awesomeProject/internal/service"
	"awesomeProject/internal/service/rabbitmq"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=10.1.1.136 user=postgres dbname=postgres port=5432 password=gsm200818534 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}
	model.InitDB(db)
	if err := db.AutoMigrate(&model.User{}, &model.BarkToken{}, &model.Threshold{}, &model.DistanceThreshold{}); err != nil {
		panic("表迁移失败: " + err.Error())
	}
	redis.InitRedis()
	go service.Hub.Run()
	if err := service.InitMQTT(); err != nil {
		log.Fatalf("MQTT 初始化失败: %v", err)
	}
	// 初始化 RabbitMQ 连接
	if err := rabbitmq.Init("60.205.140.163", "rabbitmq", "rabbitmq"); err != nil {
		log.Printf("RabbitMQ 初始化失败: %v", err)
		// 非致命，继续运行（视业务决定是否要 panic）
	} else {
		// 初始化成功 → 启动消费者（常驻）
		go rabbitmq.StartSensorDataConsumer()
		log.Println("RabbitMQ 消费者已启动")
	}

	r := gin.Default()

	// 设置可信代理，解决 Gin 警告
	err = r.SetTrustedProxies([]string{"192.168.18.0/24", "127.0.0.1"})
	if err != nil {
		log.Printf("设置可信代理失败: %v", err)
	}

	r.Use(cors.New(cors.Config{ // 解决了跨域问题
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	routes.RegisterRoutes(r, db)
	go func() {
		time.Sleep(3 * time.Second)
		service.BroadcastToWS("test/topic", `{"message": "hello from server"}`)
	}()

	r.Run("0.0.0.0:8001")
}
