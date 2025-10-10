package routes

import (
	"awesomeProject/internal/controllers"
	"awesomeProject/internal/middleware"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	userHandler := controllers.NewUserHandler(userService)

	publicAPI := r.Group("/api")
	{
		publicAPI.POST("/register", userHandler.Register)
		publicAPI.POST("/login", userHandler.Login)
	}

	// 受保护路由组（需认证）
	protectedAPI := r.Group("/api").Use(middleware.AuthMiddleware())
	{
		protectedAPI.POST("/logout", userHandler.Logout)
		protectedAPI.PUT("/update", userHandler.Update)
		protectedAPI.PUT("/modify", userHandler.ModifyPW)
	}
}
