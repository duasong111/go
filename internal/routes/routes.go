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
	sseService := service.NewSSEService(userService)
	userHandler := controllers.NewUserHandler(userService)
	sseHandler := controllers.NewSSEController(sseService) // 新开的SSE服务
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
		publicAPI.GET("/sse", sseHandler.SSEHandler)                                   // SSE传输数据
		r.GET("/esp32/data", controllers.WsHandler)                                    // ws连接
		publicAPI.POST("/device/rgb", controllers.ControlLed)                          // 控制RGB小灯
		publicAPI.POST("/device/screen/text", controllers.SendScreenText)              // 控制屏幕文字
		publicAPI.POST("/device/screen/image", controllers.UploadAndSendImage)         // 上传图片
		publicAPI.POST("/device/screen/image_url", controllers.SendScreenImageFromUrl) //使用图片的url
		publicAPI.POST("/device/buzzer", controllers.ControlBuzzer)                    // 蜂鸣器
		publicAPI.POST("/device/bark_alert", controllers.ControlBuzzer)
	}
}
