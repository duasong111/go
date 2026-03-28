package middleware

import (
	"awesomeProject/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 用户注册且用户已激活才可以控制设备
func DeviceActivationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求中获取设备ID
		var deviceID string

		// 尝试从JSON body中获取device_id
		var req struct {
			DeviceID string `json:"device_id"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.DeviceID != "" {
			deviceID = req.DeviceID
		}

		// 如果body中没有，尝试从form中获取
		if deviceID == "" {
			deviceID = c.PostForm("device_id")
		}

		// 如果还是没有，尝试从query参数获取
		if deviceID == "" {
			deviceID = c.Query("device_id")
		}

		// 如果没有提供设备ID，直接放行（兼容旧接口）
		if deviceID == "" {
			c.Next()
			return
		}

		// 检查设备是否已激活
		var factoryConfig model.DeviceFactoryConfig
		if err := model.DB.Where("device_id = ?", deviceID).First(&factoryConfig).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "设备不存在",
			})
			c.Abort()
			return
		}

		if !factoryConfig.IsActivated {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "设备未激活，请先激活设备",
			})
			c.Abort()
			return
		}

		// 设备已激活，继续处理请求
		c.Set("device_id", deviceID)
		c.Set("device_activated", true)
		c.Next()
	}
}
