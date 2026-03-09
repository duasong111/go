package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"awesomeProject/internal/model"
	"awesomeProject/internal/redis"
	"awesomeProject/internal/service"
	"awesomeProject/pkg"

	"github.com/gin-gonic/gin"
)

func AcceptBarkToken(c *gin.Context) {
	var req pkg.BarkTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取完整用户信息
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户不存在",
		})
		return
	}

	// 保存或更新 Token（一个用户通常覆盖旧的）
	tokenRecord := model.BarkToken{
		UserID:   user.ID,
		Token:    req.Token,
		Device:   req.Device,
		IsActive: true,
	}

	result := model.DB.Where("user_id = ?", user.ID).
		Assign(&tokenRecord).
		FirstOrCreate(&model.BarkToken{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "保存 Bark Token 失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Bark Token 已保存",
	})
}

// 2. 接收并保存阈值配置
func AcceptThreshold(c *gin.Context) {
	var req pkg.ThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取完整用户信息
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户不存在",
		})
		return
	}

	// 必须先绑定 Token
	var tokenCount int64
	model.DB.Model(&model.BarkToken{}).
		Where("user_id = ? AND is_active = true", user.ID).
		Count(&tokenCount)

	if tokenCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请先绑定有效的 Bark Token",
		})
		return
	}

	// 默认报警间隔 5 分钟
	if req.AlertSeconds <= 0 {
		req.AlertSeconds = 300
	}

	var threshold model.Threshold
	result := model.DB.Where("user_id = ? AND device_id = ?", user.ID, req.DeviceID).
		First(&threshold)

	if result.Error != nil {
		// 创建新记录
		threshold = model.Threshold{
			UserID:        user.ID,
			DeviceID:      req.DeviceID,
			TempMax:       req.TempMax,
			TempMin:       req.TempMin,
			HumidityMax:   req.HumidityMax,
			HumidityMin:   req.HumidityMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存阈值失败: " + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"temp_max":       req.TempMax,
			"temp_min":       req.TempMin,
			"humidity_max":   req.HumidityMax,
			"humidity_min":   req.HumidityMin,
			"alert_interval": req.AlertSeconds,
			"is_active":      req.IsActive,
		}
		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新阈值失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "阈值配置已保存",
	})
}

// 实践中给设备更新唯一的device_id 并在出厂编号中进行注明

func BarkAlert(c *gin.Context) {
	var req pkg.DeviceAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		//log.Printf("[BarkAlert] 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}
	//log.Printf("[BarkAlert] 收到设备上报数据: device_id=%s, temperature=%.2f, humidity=%.2f", req.DeviceID, req.Temperature, req.Humidity)

	var threshold model.Threshold
	if err := model.DB.Where("device_id = ?", req.DeviceID).First(&threshold).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "未配置阈值，跳过报警",
		})
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警已禁用，跳过报警",
		})
		return
	}

	// 收集报警信息
	var alertItems []string
	if threshold.TempMax != nil && req.Temperature > *threshold.TempMax {
		alertItems = append(alertItems, fmt.Sprintf("温度过高: %.1f > %.1f", req.Temperature, *threshold.TempMax))
	}
	if threshold.TempMin != nil && req.Temperature < *threshold.TempMin {
		alertItems = append(alertItems, fmt.Sprintf("温度过低: %.1f < %.1f", req.Temperature, *threshold.TempMin))
	}
	if threshold.HumidityMax != nil && req.Humidity > *threshold.HumidityMax {
		alertItems = append(alertItems, fmt.Sprintf("湿度过高: %.1f > %.1f", req.Humidity, *threshold.HumidityMax))
	}
	if threshold.HumidityMin != nil && req.Humidity < *threshold.HumidityMin {
		alertItems = append(alertItems, fmt.Sprintf("湿度过低: %.1f < %.1f", req.Humidity, *threshold.HumidityMin))
	}

	if len(alertItems) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "指标正常",
		})
		return
	}

	alertMessage := strings.Join(alertItems, "；")

	cooldownKey := fmt.Sprintf("alert:cooldown:%s:%s", req.DeviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("Redis SetNX 错误: %v", err)
		// Redis 失败时仍允许推送（避免漏报）
	} else if !setOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警冷却中",
		})
		return
	}

	// 获取用户 Bark Token（取第一个活跃的）
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).
		First(&barkToken).Error; err != nil {
		//log.Printf("未找到活跃 Bark Token: user_id=%d, err=%v", threshold.UserID, err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无可用 Bark Token",
		})
		return
	}

	// 异步推送 Bark
	go func() {
		title := fmt.Sprintf("设备[%s] 预警", req.DeviceID)
		body := alertMessage
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(body),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("Bark 请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Bark 返回非 200: %d", resp.StatusCode)
		} else {
			log.Printf("Bark 推送成功: device=%s, alert=%s", req.DeviceID, alertMessage)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "报警已触发并推送",
	})
}

// 4. 设置距离监测阈值（独立接口）
func AcceptDistanceThreshold(c *gin.Context) {
	var req pkg.DistanceThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 必须先绑定 Token
	var tokenCount int64
	model.DB.Model(&model.BarkToken{}).
		Where("user_id = ? AND is_active = true", userID).
		Count(&tokenCount)

	if tokenCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请先绑定有效的 Bark Token",
		})
		return
	}

	// 默认报警间隔 5 分钟
	if req.AlertSeconds <= 0 {
		req.AlertSeconds = 300
	}

	var threshold model.DistanceThreshold
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).
		First(&threshold)

	if result.Error != nil {
		// 创建新记录
		threshold = model.DistanceThreshold{
			UserID:        userID,
			DeviceID:      req.DeviceID,
			DistanceMin:   req.DistanceMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存距离阈值失败: " + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"distance_min":   req.DistanceMin,
			"alert_interval": req.AlertSeconds,
			"is_active":      req.IsActive,
		}
		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新距离阈值失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "距离监测阈值已保存",
	})
}

func DistanceAlert(c *gin.Context) {
	var req pkg.DistanceAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}
	//log.Printf("[DistanceAlert] 收到距离上报: device_id=%s, distance=%.2f", req.DeviceID, req.Distance)

	// 查找该设备的距离阈值配置
	var threshold model.DistanceThreshold
	if err := model.DB.Where("device_id = ?", req.DeviceID).First(&threshold).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "未配置距离阈值，跳过报警",
		})
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "距离报警已禁用，跳过报警",
		})
		return
	}

	if threshold.DistanceMin == nil || req.Distance >= *threshold.DistanceMin {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "距离正常",
		})
		return
	}

	alertMessage := fmt.Sprintf("距离过近: %.1f < %.1f", req.Distance, *threshold.DistanceMin)
	//log.Printf("[DistanceAlert] 触发报警: %s", alertMessage)

	cooldownKey := fmt.Sprintf("distance_alert:cooldown:%s:%s", req.DeviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("[DistanceAlert] Redis SetNX 错误: %v", err)
	} else if !setOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警冷却中",
		})
		return
	}

	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).
		First(&barkToken).Error; err != nil {
		log.Printf("[DistanceAlert] 未找到活跃 Bark Token: user_id=%d, err=%v", threshold.UserID, err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无可用 Bark Token",
		})
		return
	}

	go func() {
		title := fmt.Sprintf("设备[%s] 距离预警", req.DeviceID)
		body := alertMessage
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(body),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("[DistanceAlert] Bark 请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[DistanceAlert] Bark 返回非 200: %d", resp.StatusCode)
		} else {
			log.Printf("[DistanceAlert] Bark 推送成功: device=%s, alert=%s", req.DeviceID, alertMessage)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "距离报警已触发并推送",
	})
}

func SensorData(c *gin.Context) {
	var req pkg.SensorDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[SensorData] 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}
	// 创建通道来接收两个处理结果
	tempHumidityDone := make(chan bool, 1)
	distanceDone := make(chan bool, 1)

	// 异步处理温湿度报警
	go func() {
		processTempHumidityAlert(req.DeviceID, req.Temperature, req.Humidity)
		tempHumidityDone <- true
	}()

	// 异步处理距离报警
	go func() {
		processDistanceAlert(req.DeviceID, req.Distance)
		distanceDone <- true
	}()

	// 等待两个处理完成（带超时）
	select {
	case <-tempHumidityDone:
	case <-time.After(5 * time.Second):
	}

	select {
	case <-distanceDone:
	case <-time.After(5 * time.Second):
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "传感器数据已接收并处理",
	})
}

// 处理温湿度报警（内部函数）
func processTempHumidityAlert(deviceID string, temperature, humidity float64) {
	//log.Printf("[processTempHumidityAlert] 开始处理: device_id=%s, temperature=%.2f, humidity=%.2f", deviceID, temperature, humidity)

	// 查找该设备的阈值配置
	var threshold model.Threshold
	if err := model.DB.Where("device_id = ?", deviceID).First(&threshold).Error; err != nil {
		// 未配置阈值，正常情况，不打印错误日志
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		// 用户已禁用报警，正常情况，不打印错误日志
		return
	}

	// 收集报警信息
	var alertItems []string
	if threshold.TempMax != nil && temperature > *threshold.TempMax {
		alertItems = append(alertItems, fmt.Sprintf("温度过高: %.1f > %.1f", temperature, *threshold.TempMax))
	}
	if threshold.TempMin != nil && temperature < *threshold.TempMin {
		alertItems = append(alertItems, fmt.Sprintf("温度过低: %.1f < %.1f", temperature, *threshold.TempMin))
	}
	if threshold.HumidityMax != nil && humidity > *threshold.HumidityMax {
		alertItems = append(alertItems, fmt.Sprintf("湿度过高: %.1f > %.1f", humidity, *threshold.HumidityMax))
	}
	if threshold.HumidityMin != nil && humidity < *threshold.HumidityMin {
		alertItems = append(alertItems, fmt.Sprintf("湿度过低: %.1f < %.1f", humidity, *threshold.HumidityMin))
	}

	if len(alertItems) == 0 {
		//log.Printf("[processTempHumidityAlert] 指标正常: device_id=%s", deviceID)
		return
	}

	alertMessage := strings.Join(alertItems, "；")
	//log.Printf("[processTempHumidityAlert] 触发报警: %s", alertMessage)

	// 防抖
	cooldownKey := fmt.Sprintf("alert:cooldown:%s:%s", deviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		//log.Printf("[processTempHumidityAlert] Redis错误: %v", err)
		return
	}
	if !setOK {
		//log.Printf("[processTempHumidityAlert] 冷却中: device_id=%s", deviceID)
		return
	}

	// 获取Bark Token并推送
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).First(&barkToken).Error; err != nil {
		//log.Printf("[processTempHumidityAlert] 无可用Bark Token: user_id=%d", threshold.UserID)
		return
	}

	// 异步推送
	go func() {
		title := fmt.Sprintf("设备[%s] 温湿度预警", deviceID)
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(alertMessage),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			//log.Printf("[processTempHumidityAlert] Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			//log.Printf("[processTempHumidityAlert] Bark推送成功: device_id=%s", deviceID)
		}
	}()
}

// 处理距离报警（内部函数）
func processDistanceAlert(deviceID string, distance float64) {
	//log.Printf("[processDistanceAlert] 开始处理: device_id=%s, distance=%.2f", deviceID, distance)

	// 查找该设备的距离阈值配置
	var threshold model.DistanceThreshold
	if err := model.DB.Where("device_id = ?", deviceID).First(&threshold).Error; err != nil {
		// 未配置距离阈值，正常情况，不打印错误日志
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		// 用户已禁用报警，正常情况，不打印错误日志
		return
	}

	// 判断是否触发报警
	if threshold.DistanceMin == nil || distance >= *threshold.DistanceMin {
		//log.Printf("[processDistanceAlert] 距离正常: device_id=%s, distance=%.2f", deviceID, distance)
		return
	}

	alertMessage := fmt.Sprintf("距离过近: %.1f < %.1f", distance, *threshold.DistanceMin)
	//log.Printf("[processDistanceAlert] 触发报警: %s", alertMessage)

	// 防抖
	cooldownKey := fmt.Sprintf("distance_alert:cooldown:%s:%s", deviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		//log.Printf("[processDistanceAlert] Redis错误: %v", err)
		return
	}
	if !setOK {
		//log.Printf("[processDistanceAlert] 冷却中: device_id=%s", deviceID)
		return
	}

	// 获取Bark Token并推送
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).First(&barkToken).Error; err != nil {
		//log.Printf("[processDistanceAlert] 无可用Bark Token: user_id=%d", threshold.UserID)
		return
	}

	// 异步推送
	go func() {
		title := fmt.Sprintf("设备[%s] 距离预警", deviceID)
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(alertMessage),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			//log.Printf("[processDistanceAlert] Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			//log.Printf("[processDistanceAlert] Bark推送成功: device_id=%s", deviceID)
		}
	}()
}

// 统一设备控制接口
func ControlDevices(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
		Controls []struct {
			Type       string `json:"type" binding:"required,oneof=buzzer led"`
			State      string `json:"state" binding:"required,oneof=on off"`
			Frequency  *int   `json:"frequency,omitempty"`
			Duration   *int   `json:"duration,omitempty"`
			Cycles     *int   `json:"cycles,omitempty"`
			Interval   *int   `json:"interval,omitempty"`
			Color      string `json:"color,omitempty"`
			Brightness *int   `json:"brightness,omitempty"`
			Mode       string `json:"mode,omitempty"`
		} `json:"controls" binding:"required,dive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 处理每个控制命令
	for _, control := range req.Controls {
		var payload map[string]interface{}

		switch control.Type {
		case "buzzer":
			payload = map[string]interface{}{
				"type":  "buzzer",
				"state": control.State,
			}
			if control.State == "on" {
				if control.Frequency != nil {
					payload["frequency"] = *control.Frequency
				} else {
					payload["frequency"] = 2000
				}
				if control.Duration != nil {
					payload["duration"] = *control.Duration
				} else {
					payload["duration"] = 500
				}
				if control.Cycles != nil {
					payload["cycles"] = *control.Cycles
				}
				if control.Interval != nil {
					payload["interval"] = *control.Interval
				}
			}

		case "led":
			payload = map[string]interface{}{
				"state": control.State,
			}
			if control.State == "on" {
				if control.Color != "" {
					payload["color"] = control.Color
				} else {
					payload["color"] = "white"
				}
				if control.Brightness != nil {
					payload["brightness"] = *control.Brightness
				} else {
					payload["brightness"] = 100
				}
				if control.Mode != "" {
					payload["mode"] = control.Mode
				}
				if control.Interval != nil {
					payload["interval"] = *control.Interval
				}
				if control.Duration != nil {
					payload["duration"] = *control.Duration
				}
			} else {
				payload["color"] = "black"
				payload["brightness"] = 0
			}
		}

		// 发送到 MQTT
		err := service.Publish("control/esp32", 0, false, payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "MQTT 命令发送失败：" + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          200,
		"message":       "设备控制命令已发送",
		"device_id":     req.DeviceID,
		"control_count": len(req.Controls),
	})
}
