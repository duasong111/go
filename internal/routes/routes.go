package routes

import (
	"awesomeProject/internal/controllers"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 因为在各层中的调用使用了工厂模式，要在此处去进行调用数据库的实例化
func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := controllers.NewUserHandler(userService)

	api := r.Group("/api")
	{
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)
		api.POST("/logout", userHandler.Logout)
	}
}
