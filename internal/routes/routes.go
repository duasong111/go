package routes

import (
	"awesomeProject/controllers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/login", controllers.Login)
		api.GET("/logout", controllers.Logout)
	}
	server := r.Group("/server")
	{
		server.GET("/hht")
	}
}
