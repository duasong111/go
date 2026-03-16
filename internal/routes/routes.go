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
	sseHandler := controllers.NewSSEController(sseService)// 公开路由（无需认证）
	publicAPI := r.Group("/api")
	{
		publicAPI.POST("/register", userHandler.Register)
		publicAPI.POST("/login", userHandler.Login)
		publicAPI.GET("/logs/search", controllers.SearchLogs)           // 搜索日志
		publicAPI.GET("/logs/stats", controllers.GetLogStats)            // 获取统计（需先实现 GetLogStats 处理器）
		publicAPI.GET("/logs/device/:device_id", controllers.GetDeviceLogs) // 获取设备日志
	} 
	
	protectedAPI := r.Group("/api").Use(middleware.AuthMiddleware())
	{
		protectedAPI.POST("/logout", userHandler.Logout)
		protectedAPI.GET("/user/info", userHandler.GetUserInfo) // 获取用户信息
		protectedAPI.PUT("/update", userHandler.Update)
		protectedAPI.PUT("/modify", userHandler.ModifyPW)
		protectedAPI.POST("/device/accept_threshold", controllers.AcceptThreshold)                  // 接收阈值
		protectedAPI.POST("/device/accept_bark_token", controllers.AcceptBarkToken)                 // 接收用户的token
		protectedAPI.POST("/device/manage_threshold", controllers.ManageThreshold)                  // 升级版：管理温湿度阈值
		protectedAPI.POST("/device/manage_distance_threshold", controllers.ManageDistanceThreshold) // 升级版：管理距离阈值
	}

	publicAPI.GET("/sse", sseHandler.SSEHandler)                                   // SSE传输数据
	r.GET("/esp32/data", controllers.WsHandler)                                    // ws连接
	publicAPI.POST("/device/rgb", controllers.ControlLed)                          // 控制RGB小灯
	publicAPI.POST("/device/screen/text", controllers.SendScreenText)              // 控制屏幕文字
	publicAPI.POST("/device/screen/image", controllers.UploadAndSendImage)         // 上传图片
	publicAPI.POST("/device/screen/image_url", controllers.SendScreenImageFromUrl) //使用图片的url
	publicAPI.POST("/device/buzzer", controllers.ControlBuzzer)                    // 蜂鸣器
	publicAPI.POST("/device/bark_alert", controllers.BarkAlert)                    // 温湿度报警（设备上报）
	publicAPI.POST("/device/distance_alert", controllers.DistanceAlert)            // 距离报警（设备上报，独立接口）
	publicAPI.POST("/device/sensor_data", controllers.SensorData)
	publicAPI.POST("/device/control_self", controllers.ControlDevices)                   // 自身设备结合
	protectedAPI.POST("/device/distance_threshold", controllers.AcceptDistanceThreshold) // 距离阈值设置（独立接口）
}
