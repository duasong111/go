package controllers

import (
	"awesomeProject/internal/model"
	"awesomeProject/pkg"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 设备离线监测配置请求
type DeviceOfflineConfigRequest struct {
	DeviceID         string `json:"device_id" binding:"required"`
	IsActive         bool   `json:"is_active"`
	EnableOnlineAlert bool   `json:"enable_online_alert,omitempty"` // 是否启用上线提醒
	MaxAlertCount    int    `json:"max_alert_count,omitempty"`     // 最大通知次数
	OfflineThreshold int    `json:"offline_threshold,omitempty"` // 离线判断阈值（秒）
	AlertInterval    int    `json:"alert_interval,omitempty"`    // 报警间隔（秒）
}

// 管理设备离线监测配置
func ManageDeviceOfflineConfig(c *gin.Context) {
	var req DeviceOfflineConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, "参数格式错误: "+err.Error())
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "请先登录")
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查询现有配置
	var config model.DeviceOfflineConfig
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&config)

	if result.Error != nil {
		// 创建新配置
		if req.OfflineThreshold <= 0 {
			req.OfflineThreshold = 300 // 默认5分钟
		}
		if req.AlertInterval <= 0 {
			req.AlertInterval = 300 // 默认5分钟
		}
		if req.MaxAlertCount <= 0 {
			req.MaxAlertCount = 3 // 默认3次
		}

		config = model.DeviceOfflineConfig{
			UserID:           userID,
			DeviceID:         req.DeviceID,
			IsActive:         req.IsActive,
			EnableOnlineAlert: req.EnableOnlineAlert,
			MaxAlertCount:    req.MaxAlertCount,
			OfflineThreshold: req.OfflineThreshold,
			AlertInterval:    req.AlertInterval,
		}

		if err := model.DB.Create(&config).Error; err != nil {
			pkg.ErrorResponse(c, http.StatusInternalServerError, "创建配置失败: "+err.Error())
			return
		}
	} else {
		// 更新现有配置
		updates := map[string]interface{}{
			"is_active":          req.IsActive,
			"enable_online_alert": req.EnableOnlineAlert,
			"max_alert_count":     req.MaxAlertCount,
			"offline_threshold":  req.OfflineThreshold,
			"alert_interval":     req.AlertInterval,
		}

		if req.OfflineThreshold <= 0 {
			updates["offline_threshold"] = 300 // 默认5分钟
		}
		if req.AlertInterval <= 0 {
			updates["alert_interval"] = 300 // 默认5分钟
		}
		if req.MaxAlertCount <= 0 {
			updates["max_alert_count"] = 3 // 默认3次
		}

		if err := model.DB.Model(&config).Updates(updates).Error; err != nil {
			pkg.ErrorResponse(c, http.StatusInternalServerError, "更新配置失败: "+err.Error())
			return
		}
	}

	pkg.RightResponse(c, gin.H{
		"config": config,
	}, "配置成功")
}

// 获取设备离线监测配置
func GetDeviceOfflineConfig(c *gin.Context) {
	deviceID := c.Query("device_id")
	if deviceID == "" {
		pkg.ErrorResponse(c, http.StatusBadRequest, "设备ID不能为空")
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "请先登录")
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查询配置
	var config model.DeviceOfflineConfig
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&config)

	if result.Error != nil {
		// 配置不存在，返回默认配置
		pkg.RightResponse(c, gin.H{
			"config": model.DeviceOfflineConfig{
				UserID:           userID,
				DeviceID:         deviceID,
				IsActive:         false,
				EnableOnlineAlert: false,
				MaxAlertCount:    3,
				OfflineThreshold: 300,
				AlertInterval:    300,
			},
		}, "配置不存在，返回默认值")
		return
	}

	pkg.RightResponse(c, gin.H{
		"config": config,
	}, "获取配置成功")
}
