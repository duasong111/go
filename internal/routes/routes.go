package routes

import (
	"awesomeProject/internal/controllers"
	"awesomeProject/internal/middleware"
	"awesomeProject/internal/repository"
	"awesomeProject/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB) {
	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)
	sseService := service.NewSSEService(userService)
	userHandler := controllers.NewUserHandler(userService)
	sseHandler := controllers.NewSSEController(sseService) // 公开路由（无需认证）

	// 提供 swagger.yaml 文件的静态访问
	r.StaticFile("/api/swagger.yaml", "./api/swagger.yaml")

	// 添加 Swagger 路由，使用本地 swagger.yaml 文件
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/swagger.yaml")))

	publicAPI := r.Group("/api")
	{
		publicAPI.POST("/register", userHandler.Register)
		publicAPI.POST("/login", userHandler.Login)
		publicAPI.GET("/logs/search", controllers.SearchLogs)               // 搜索日志
		publicAPI.GET("/logs/stats", controllers.GetLogStats)               // 获取统计（需先实现 GetLogStats 处理器）
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
	publicAPI.POST("/device/bark_alert", controllers.BarkAlert)                    // 温湿度报警（设备上报）
	publicAPI.POST("/device/distance_alert", controllers.DistanceAlert)            // 距离报警（设备上报，独立接口）
	publicAPI.POST("/device/sensor_data", controllers.SensorData)                  // 设备上传数据

	// 设备控制接口（需要认证 + 设备已激活）
	deviceControlAPI := r.Group("/api/device").Use(middleware.AuthMiddleware(), middleware.DeviceActivationMiddleware())
	{
		deviceControlAPI.POST("/rgb", controllers.ControlLed)                          // 控制RGB小灯
		deviceControlAPI.POST("/screen/text", controllers.SendScreenText)              // 控制屏幕文字
		deviceControlAPI.POST("/screen/image", controllers.UploadAndSendImage)         // 上传图片
		deviceControlAPI.POST("/screen/image_url", controllers.SendScreenImageFromUrl) // 使用图片的url
		deviceControlAPI.POST("/buzzer", controllers.ControlBuzzer)                    // 蜂鸣器
		deviceControlAPI.POST("/control_self", controllers.ControlDevices)             // 自身设备结合
	}

	// 设备管理接口（需要认证） 
	deviceManageAPI := r.Group("/api/device").Use(middleware.AuthMiddleware()) 
	{
		deviceManageAPI.POST("/distance_threshold", controllers.AcceptDistanceThreshold) // 距离阈值设置（独立接口）
		deviceManageAPI.POST("/offline_config", controllers.ManageDeviceOfflineConfig)   // 管理设备离线监测配置
		deviceManageAPI.GET("/offline_config", controllers.GetDeviceOfflineConfig)       // 获取设备离线监测配置
		deviceManageAPI.POST("/bind", controllers.BindDevice)                            // 设备绑定
		deviceManageAPI.POST("/unbind", controllers.UnbindDevice)                        // 设备解绑
		deviceManageAPI.GET("/list", controllers.GetUserDevices)                         // 获取用户设备列表
	}

	// 通知接口（需要认证）
	notificationAPI := r.Group("/api").Use(middleware.AuthMiddleware())
	{
		notificationAPI.GET("/notifications", controllers.GetUserNotifications)           // 获取用户提醒列表
		notificationAPI.PUT("/notifications/:id/read", controllers.MarkNotificationAsRead) // 标记提醒为已读
	}

}
