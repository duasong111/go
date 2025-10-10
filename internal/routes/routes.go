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

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/register", userHandler.Register)
		api.POST("/login", userHandler.Login)
		api.POST("/logout", userHandler.Logout)
	}
}
