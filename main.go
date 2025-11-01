package main

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/routes"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=43.136.37.113 user=admin dbname=pg_go port=5432 password=PGPass123! sslmode=disable"
	// dsn := "host=192.168.1.88 user=postgres dbname=intellicamera port=5432 password=gsm200818534 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败: " + err.Error())
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		panic("表迁移失败: " + err.Error())
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
	r.Run("0.0.0.0:8000")
}
