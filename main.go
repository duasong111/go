package main

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/elasticsearch"
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
	// 加载配置文件
	if err := config.LoadConfig("./config/config.yaml"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(config.AppConfig.Database.GetDSN()), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}
	model.InitDB(db)
	if err := db.AutoMigrate(&model.User{}, &model.BarkToken{}, &model.Threshold{}, &model.DistanceThreshold{}, &model.DeviceOfflineConfig{}); err != nil {
		panic("表迁移失败: " + err.Error())
	}

	// 初始化 Redis
	redis.InitRedis()

	// 启动 WebSocket Hub
	go service.Hub.Run()

	// 初始化 MQTT
	if err := service.InitMQTT(); err != nil {
		log.Fatalf("MQTT 初始化失败: %v", err)
	}

	// 初始化 RabbitMQ 连接
	if err := rabbitmq.Init(
		config.AppConfig.RabbitMQ.Host,
		config.AppConfig.RabbitMQ.User,
		config.AppConfig.RabbitMQ.Password,
	); err != nil {
		log.Printf("RabbitMQ 初始化失败: %v", err)
	} else {
		go rabbitmq.StartSensorDataConsumer()
		log.Println("RabbitMQ 消费者已启动")
	}

	// 启动设备离线监测服务
	service.StartDeviceOfflineMonitor()

	// 创建 Gin 引擎
	r := gin.Default()

	// 设置可信代理
	err = r.SetTrustedProxies([]string{"192.168.18.0/24", "127.0.0.1"})
	if err != nil {
		log.Printf("设置可信代理失败: %v", err)
	}

	// 配置 CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     config.AppConfig.CORS.AllowOrigins,
		AllowMethods:     config.AppConfig.CORS.AllowMethods,
		AllowHeaders:     config.AppConfig.CORS.AllowHeaders,
		ExposeHeaders:    config.AppConfig.CORS.ExposeHeaders,
		AllowCredentials: config.AppConfig.CORS.AllowCredentials,
		MaxAge:           time.Duration(config.AppConfig.CORS.MaxAge) * time.Second,
	}))

	// 注册路由
	routes.RegisterRoutes(r, db)

	// 测试 WebSocket
	go func() {
		time.Sleep(3 * time.Second)
		service.BroadcastToWS("test/topic", `{"message": "hello from server"}`)
	}()

	// 初始化 Elasticsearch
	if err := elasticsearch.Init(); err != nil {
		log.Printf("Elasticsearch 初始化失败: %v", err)
		// 非致命，继续运行
	}

	// 启动服务器
	log.Printf("服务器启动在端口: %d", config.AppConfig.Server.Port)
	r.Run(":8000")
}
