package main

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/routes"
	"awesomeProject/internal/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

func main() {
	dsn := "host=60.205.140.163 user=user_yrh7kC dbname=go_pg port=5432 password=password_asY8fN sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		panic("表迁移失败: " + err.Error())
	}
	go service.Hub.Run()
	if err := service.InitMQTT(); err != nil {
		log.Fatalf("MQTT 初始化失败: %v", err)
	}
	r := gin.Default()

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

	r.Run("0.0.0.0:8000")
}
