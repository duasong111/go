package routes

import (
	"awesomeProject/internal/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/register", controllers.Register)
		api.GET("/login", controllers.Login)
		api.GET("/logout", controllers.Logout)
	}
}
