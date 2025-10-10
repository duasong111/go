package main

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/routes"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 好处是可以无障碍的去切换数据库

func main() {
	dsn := "host=192.168.1.3 user=postgres dbname=intellicamera port=5432 password=gsm200818534 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		panic("表迁移失败: " + err.Error())
	}
	r := gin.Default()
	routes.RegisterRoutes(r, db)
	r.Run("0.0.0.0:8080")
}
